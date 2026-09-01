package auth

import (
	"back-rex-common/pkg/services"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"context"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

// logRefusRefresh trace tout refus de /auth/refresh en Warn : les 400 partent
// sinon en Debug (cf. services.RenderError), invisibles en production — un
// utilisateur déconnecté « sans raison » ne laissait aucune trace exploitable.
// Le jeton lui-même n'est JAMAIS logué, même haché.
func logRefusRefresh(ctx context.Context, r *http.Request, raison string, attrs ...any) {
	// ip + referer : instrumentation temporaire (branche pb), pour corréler
	// chaque refus avec l'access log nginx et la page qui a déclenché l'appel.
	args := append([]any{"raison", raison, "ip", ipClient(r), "referer", r.Referer(),
		"user_agent", r.UserAgent()}, attrs...)
	slog.WarnContext(ctx, "refresh refusé", args...)
}

// logJetonInconnu qualifie un « Refresh token inconnu » : le 400 rendu au
// client reste volontairement opaque, mais le log distingue les causes. Le
// jeton reçu est un JWT : sa signature dit s'il vient bien de nous, et ses
// claims (sub, session_id, version, exp) donnent de quoi corréler.
//
// La session étant stable à travers les rotations (generateRefreshToken), une
// lecture par session tranche entre « déjà consommé » (la session vit avec un
// jeton plus récent : replay ou course perdue) et « session fermée » (logout,
// révocation). Lecture SEULE : la révocation de famille sur replay reste le
// lot token_version.
func logJetonInconnu(ctx context.Context, r *http.Request, q *Queries, secret string, brut string) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	tok, err := parser.Parse(brut, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		logRefusRefresh(ctx, r, "jeton étranger ou corrompu (signature invalide)")
		return
	}

	claims, _ := tok.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	session, _ := claims["session_id"].(string)
	version, _ := claims["version"].(float64)

	exp, expErr := claims.GetExpirationTime()
	attrs := []any{"user", sub, "session", session, "version_jeton", int(version)}
	if expErr == nil && exp != nil {
		attrs = append(attrs, "exp_jeton", exp.Time.Format(time.RFC3339))
		if time.Now().After(exp.Time) {
			logRefusRefresh(ctx, r, "jeton périmé, vraisemblablement purgé (CleanUpTokens)", attrs...)
			return
		}
	}

	enBase, err := q.GetRefreshTokenBySession(ctx, session)
	if err == nil {
		attrs = append(attrs, "version_en_base", enBase.TokenVersion.Int32)
		// Instrumentation temporaire (branche pb) : situer le jeton rejoué par
		// rapport au vivant. Depuis la réémission idempotente, le prédécesseur
		// direct est RÉÉMIS, pas refusé — recu_est_prev_direct=true ici ne
		// devrait plus apparaître que pour un JWT périmé ; false = la chaîne a
		// avancé d'au moins deux crans (vrai rejeu, ou storage très ancien).
		attrs = append(attrs,
			"jeton_recu_hash", prefixeHash(hashToken(brut)),
			"jeton_en_base_hash", prefixeHash(enBase.Token),
			"derniere_rotation", enBase.CreatedAt.Time.Format(time.RFC3339))
		if enBase.PrevToken.Valid {
			attrs = append(attrs,
				"prev_en_base_hash", prefixeHash(enBase.PrevToken.String),
				"recu_est_prev_direct", hashToken(brut) == enBase.PrevToken.String)
		}
		if enBase.PrevConsumedAt.Valid {
			attrs = append(attrs,
				"prev_consomme_depuis", time.Since(enBase.PrevConsumedAt.Time).Round(time.Second).String())
		}
		logRefusRefresh(ctx, r,
			"jeton déjà consommé : la session porte un jeton plus récent (replay après rotation, ou course perdue)",
			attrs...)
		return
	}
	logRefusRefresh(ctx, r, "session fermée (logout ou révocation)", attrs...)
}

// jetonReemissible tente la réémission idempotente sur un jeton introuvable
// par ConsumeRefreshToken.
//
// La rotation stricte fabrique un jeton ORPHELIN quand la réponse de
// /auth/refresh n'atteint jamais le client : le serveur a tourné v→v+1, mais
// le client (timeout axios, fermeture ou reload qui avorte la requête en vol)
// ne connaît que v. Il le rejoue — parfois des minutes ou des heures plus
// tard, à l'ouverture suivante de l'application — recevait « jeton déjà
// consommé » et se déconnectait. Constaté en production les 30/08 et
// 01/09/2026 ; la fenêtre de grâce de 60 s essayée entre-temps couvrait le
// retry immédiat mais pas le retour tardif, et aucune fenêtre TEMPORELLE ne
// peut le couvrir : c'est l'idempotence qu'il faut.
//
// Si le jeton présenté est un JWT à nous, non périmé, et le prédécesseur
// DIRECT du jeton vivant (prev_token), la ligne vivante est retournée :
// l'appelant re-fabrique ce jeton courant À L'IDENTIQUE
// (regenereRefreshToken) et le rend avec un access token neuf — la même
// réponse que celle qui s'est perdue, sans rien écrire en base, rejouable à
// volonté. La détection de rejeu garde son sens avec un cran de retard : dès
// que le jeton courant est consommé à son tour, prev_token avance et
// l'ancien prédécesseur retombe mécaniquement sur ErrNoRows — le refus, et à
// terme la révocation de famille du lot token_version, reprennent leurs
// droits.
func jetonReemissible(ctx context.Context, q *Queries, jwtCfg services.JWTConfig, brut string, hash string) (RefreshToken, error) {
	// Signature et expiration vérifiées sur le JWT présenté lui-même : son
	// éventuelle ligne en base a été consommée, il ne reste que lui pour en
	// répondre. Un jeton périmé ou étranger n'ouvre droit à rien.
	tok, err := jwt.Parse(brut, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue %v", t.Header["alg"])
		}
		return []byte(jwtCfg.Secret), nil
	})
	if err != nil || !tok.Valid {
		return RefreshToken{}, pgx.ErrNoRows
	}
	return q.GetRefreshTokenByPrev(ctx, services.ToPgText(hash))
}

// ── Instrumentation temporaire (branche pb) ────────────────────────────────
//
// Trace CHAQUE passage dans /auth/refresh, entrée comme sortie, pour
// comprendre les déconnexions résiduelles en production. Entorse assumée à la
// règle « jamais le jeton dans les logs » : le HACHAGE est logué TRONQUÉ à 12
// hex — assez pour corréler avec les colonnes token / prev_token de
// refresh_tokens, inutilisable pour s'authentifier (le jeton en clair n'est
// pas dérivable de son SHA-256). À RETIRER une fois le diagnostic posé.

// prefixeHash rend les 12 premiers hex du hachage déjà calculé d'un jeton.
func prefixeHash(hash string) string {
	if len(hash) < 12 {
		return hash
	}
	return hash[:12]
}

// ipClient remonte l'adresse réelle du client posée par le nginx du conteneur
// (X-Real-IP), pour corréler avec son access log ; à défaut, l'adresse TCP.
func ipClient(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// attrsJetonRecu décode le jeton présenté SANS le valider (même périmé ou
// étranger, on veut savoir qui prétend quoi) et rend les attributs de log :
// hachage tronqué, user, session, version, expiration.
func attrsJetonRecu(secret string, brut string) []any {
	attrs := []any{"jeton_hash", prefixeHash(hashToken(brut))}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	tok, err := parser.Parse(brut, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return append(attrs, "claims", "illisibles ou signature invalide")
	}
	claims, _ := tok.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	session, _ := claims["session_id"].(string)
	version, _ := claims["version"].(float64)
	attrs = append(attrs, "user", sub, "session", session, "version_jeton", int(version))
	if exp, expErr := claims.GetExpirationTime(); expErr == nil && exp != nil {
		attrs = append(attrs, "exp_jeton", exp.Time.Format(time.RFC3339))
	}
	return attrs
}

// ── Fin de l'instrumentation temporaire (helpers) ──────────────────────────

// RefreshAccessToken tourne le refresh token (rotation stricte, usage unique,
// avec réémission idempotente du courant pour réponse perdue — voir
// jetonReemissible).
//
// La consommation de l'ancien jeton est ATOMIQUE (DELETE … RETURNING) : de deux
// requêtes concurrentes portant le même jeton, une seule obtient la ligne,
// l'autre reçoit un 400 avant d'avoir rien créé. L'ancien SELECT puis DELETE
// laissait les deux passer et créait deux sessions.
//
// La transaction enveloppe consommation ET insertion : si l'INSERT échoue, le
// rollback restitue l'ancien jeton au lieu de laisser le client sans session.
// Elle porte le contexte de la requête : un client parti n'engage rien.
func RefreshAccessToken(w http.ResponseWriter, r *http.Request, jwtConfig services.JWTConfig) {
	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)

	var body refreshBody
	if err := render.DecodeJSON(r.Body, &body); err != nil || body.RefreshToken == "" {
		logRefusRefresh(ctx, r, "refresh token manquant dans le corps de la requête")
		services.InvalidRequestError(w, r, "refresh token manquant", services.NO_INFORMATION, nil)
		return
	}

	// Instrumentation temporaire (branche pb) : l'ENTRÉE de chaque refresh,
	// avant toute décision — qui présente quoi, depuis où.
	debut := time.Now()
	slog.InfoContext(ctx, "refresh reçu",
		append(attrsJetonRecu(jwtConfig.Secret, body.RefreshToken),
			"ip", ipClient(r), "referer", r.Referer(), "user_agent", r.UserAgent())...)

	tx, err := pgCtx.Db.Begin(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	defer tx.Rollback(ctx)
	qtx := New(pgCtx.Db).WithTx(tx)

	hash := hashToken(body.RefreshToken)
	reemission := false
	oldRefreshToken, err := qtx.ConsumeRefreshToken(ctx, hash)
	if err == pgx.ErrNoRows {
		// Déjà consommé ? Peut-être une réponse de rotation perdue : si le
		// jeton présenté est le prédécesseur direct du jeton vivant, on rendra
		// au client LA MÊME réponse que celle qui s'est perdue (réémission
		// idempotente, voir jetonReemissible) — oldRefreshToken est alors la
		// ligne VIVANTE, pas une ligne consommée.
		oldRefreshToken, err = jetonReemissible(ctx, qtx, jwtConfig, body.RefreshToken, hash)
		reemission = err == nil
	}
	if err == pgx.ErrNoRows {
		// Jeton inconnu, plus ancien que le prédécesseur du vivant (la chaîne
		// a avancé : vrai rejeu), ou périmé : le log qualifie, le client
		// reçoit un 400 volontairement opaque.
		// Lecture via le pool : la transaction sera annulée par le defer.
		logJetonInconnu(ctx, r, New(pgCtx.Db), jwtConfig.Secret, body.RefreshToken)
		services.InvalidRequestError(w, r, "Refresh token inconnu", services.NO_INFORMATION, nil)
		return
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// Refus AVANT commit : le rollback du defer restitue la ligne, un jeton
	// révoqué ou expiré reste donc visible en base, comme avant ce correctif.
	if oldRefreshToken.Revoked {
		logRefusRefresh(ctx, r, "jeton révoqué",
			"user", oldRefreshToken.UserID, "session", oldRefreshToken.Session)
		services.InvalidRequestError(w, r, "token has been revoked", services.NO_INFORMATION, nil)
		return
	}
	if time.Now().After(oldRefreshToken.ExpiresAt.Time) {
		logRefusRefresh(ctx, r, "jeton expiré",
			"user", oldRefreshToken.UserID, "session", oldRefreshToken.Session,
			"expire_depuis", time.Since(oldRefreshToken.ExpiresAt.Time).Round(time.Minute).String())
		services.InvalidRequestError(w, r, "token has been expires", services.NO_INFORMATION, nil)
		return
	}

	user, err := qtx.GetUserById(ctx, oldRefreshToken.UserID)
	if err == pgx.ErrNoRows {
		logRefusRefresh(ctx, r, "utilisateur du jeton absent de la base",
			"user", oldRefreshToken.UserID)
		services.InvalidRequestError(w, r, "Utilisateur inconnu", services.NO_INFORMATION, nil)
		return
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if user.Blame.Valid && user.Blame.Bool {
		logRefusRefresh(ctx, r, "utilisateur banni", "user", oldRefreshToken.UserID)
		services.AuthorizationError(w, r, "Utilisateur banni", services.NO_INFORMATION, nil)
		return
	}

	claims := jwt.MapClaims{"roles": strings.Join(user.Roles, ",")}
	subject := strconv.Itoa(int(oldRefreshToken.UserID))

	if reemission {
		// La réponse perdue, rendue à l'identique : le jeton courant
		// reconstruit depuis la ligne vivante, accompagné d'un access token
		// neuf. RIEN n'est écrit en base — rejouable à volonté, deux rejeux
		// concurrents obtiennent la même chose.
		refreshBrut, err := regenereRefreshToken(&oldRefreshToken, jwtConfig)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		accessToken, err := generateAccessToken(subject, jwtConfig, &claims, oldRefreshToken.Session)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		// En Warn : chaque passage ici est une réponse perdue côté client
		// (timeout, fermeture ou reload en vol) — un signal à compter en
		// production.
		attrs := []any{"user", oldRefreshToken.UserID, "session", oldRefreshToken.Session,
			"version_reemise", oldRefreshToken.TokenVersion.Int32}
		if oldRefreshToken.PrevConsumedAt.Valid {
			attrs = append(attrs, "rotation_perdue_depuis",
				time.Since(oldRefreshToken.PrevConsumedAt.Time).Round(time.Second).String())
		}
		slog.WarnContext(ctx, "refresh réémis à l'identique : rotation précédente jamais reçue par le client", attrs...)

		// Instrumentation temporaire (branche pb) : la SORTIE, chemin réémission.
		slog.InfoContext(ctx, "refresh accordé",
			"user", oldRefreshToken.UserID,
			"session", oldRefreshToken.Session,
			"version_emise", oldRefreshToken.TokenVersion.Int32,
			"reemission", true,
			"jeton_presente_hash", prefixeHash(hash),
			"jeton_emis_hash", prefixeHash(oldRefreshToken.Token),
			"exp_emise", oldRefreshToken.ExpiresAt.Time.Format(time.RFC3339),
			"ip", ipClient(r),
			"duree", time.Since(debut).Round(time.Millisecond).String())

		render.JSON(w, r, map[string]string{
			"accessToken":  accessToken.Token,
			"refreshToken": refreshBrut,
		})
		return
	}

	tokenPaire, err := genereTokenPaire(jwtConfig, &oldRefreshToken, &claims, subject)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	now := time.Now()
	err = qtx.CreateRefreshToken(ctx, CreateRefreshTokenParams{
		Userid:       oldRefreshToken.UserID,
		Token:        hashToken(tokenPaire.RefreshTokenInfo.Token),
		Session:      tokenPaire.RefreshTokenInfo.Session,
		TokenVersion: services.ToPgInt4(tokenPaire.RefreshTokenInfo.Version),
		Expire:       services.ToPgTimestamptz(&tokenPaire.RefreshTokenInfo.Expiration),
		Created:      services.ToPgTimestamptz(&now),
		Revoked:      false,
		// Ancrage de la réémission idempotente : tant que le jeton créé ici
		// n'est pas consommé à son tour, son prédécesseur reste échangeable
		// contre lui, à l'identique (cf. jetonReemissible).
		PrevToken:      services.ToPgText(hash),
		PrevConsumedAt: services.ToPgTimestamptz(&now),
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// Instrumentation temporaire (branche pb) : la SORTIE de chaque refresh
	// réussi — ce qui a été consommé, ce qui est délivré, par quel chemin.
	slog.InfoContext(ctx, "refresh accordé",
		"user", oldRefreshToken.UserID,
		"session", oldRefreshToken.Session,
		"version_consommee", oldRefreshToken.TokenVersion.Int32,
		"version_emise", tokenPaire.RefreshTokenInfo.Version,
		"reemission", false,
		"jeton_consomme_hash", prefixeHash(hash),
		"jeton_emis_hash", prefixeHash(hashToken(tokenPaire.RefreshTokenInfo.Token)),
		"exp_emise", tokenPaire.RefreshTokenInfo.Expiration.Format(time.RFC3339),
		"ip", ipClient(r),
		"duree", time.Since(debut).Round(time.Millisecond).String())

	render.JSON(w, r, map[string]string{
		"accessToken":  tokenPaire.AccessToken.Token,
		"refreshToken": tokenPaire.RefreshTokenInfo.Token,
	})
}

type refreshBody struct {
	RefreshToken string `json:"refreshToken"`
}
