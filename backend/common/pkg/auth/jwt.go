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
	args := append([]any{"raison", raison, "user_agent", r.UserAgent()}, attrs...)
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
		logRefusRefresh(ctx, r,
			"jeton déjà consommé : la session porte un jeton plus récent (replay après rotation, ou course perdue)",
			append(attrs, "version_en_base", enBase.TokenVersion.Int32)...)
		return
	}
	logRefusRefresh(ctx, r, "session fermée (logout ou révocation)", attrs...)
}

// refreshGraceWindow : durée pendant laquelle un jeton déjà consommé reste
// échangeable une nouvelle fois (voir jetonEnGrace). Assez long pour couvrir
// un timeout client (8 s) suivi d'un rechargement de page sur réseau mobile,
// assez court pour que la détection de rejeu garde son sens.
const refreshGraceWindow = 60 * time.Second

// jetonEnGrace tente la fenêtre de grâce sur un jeton introuvable par
// ConsumeRefreshToken.
//
// La rotation stricte fabrique un jeton ORPHELIN quand la réponse de
// /auth/refresh n'atteint jamais le client : le serveur a tourné v→v+1, mais
// le client (timeout axios, reload qui avorte la requête en vol, réseau
// mobile) ne connaît que v. Il le rejoue, recevait « jeton déjà consommé » et
// se déconnectait — constaté en production le 30/08/2026, deux POST espacés
// d'exactement 8 s (le timeout client).
//
// Si le jeton présenté est le prédécesseur DIRECT du jeton vivant et que sa
// consommation date de moins de refreshGraceWindow, la ligne vivante est
// retournée : l'appelant la re-tourne EN PLACE (GraceRotateRefreshToken), ce
// qui invalide le successeur jamais reçu et en délivre un nouveau. Au-delà de
// la fenêtre, ErrNoRows : le refus — et, à terme, la révocation de famille du
// lot token_version — reprend ses droits.
func jetonEnGrace(ctx context.Context, qtx *Queries, hash string) (RefreshToken, error) {
	vivant, err := qtx.GetRefreshTokenByPrev(ctx, services.ToPgText(hash))
	if err != nil {
		return RefreshToken{}, err
	}
	if !vivant.PrevConsumedAt.Valid || time.Since(vivant.PrevConsumedAt.Time) > refreshGraceWindow {
		return RefreshToken{}, pgx.ErrNoRows
	}
	return vivant, nil
}

// RefreshAccessToken tourne le refresh token (rotation stricte, usage unique,
// avec une courte fenêtre de grâce pour réponse perdue — voir jetonEnGrace).
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

	tx, err := pgCtx.Db.Begin(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	defer tx.Rollback(ctx)
	qtx := New(pgCtx.Db).WithTx(tx)

	hash := hashToken(body.RefreshToken)
	grace := false
	oldRefreshToken, err := qtx.ConsumeRefreshToken(ctx, hash)
	if err == pgx.ErrNoRows {
		// Déjà consommé il y a un instant ? Peut-être une réponse de rotation
		// perdue : la fenêtre de grâce rend alors la ligne VIVANTE, verrouillée
		// FOR UPDATE, que l'on re-tournera en place au lieu d'en insérer une.
		oldRefreshToken, err = jetonEnGrace(ctx, qtx, hash)
		grace = err == nil
	}
	if err == pgx.ErrNoRows {
		// Jeton inconnu, tourné hors fenêtre de grâce, ou perdu face à une
		// requête concurrente : le log qualifie, le client reçoit un 400
		// volontairement opaque.
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

	tokenPaire, err := genereTokenPaire(jwtConfig, &oldRefreshToken, &claims, strconv.Itoa(int(oldRefreshToken.UserID)))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	now := time.Now()
	if grace {
		// Re-rotation en place : le successeur jamais reçu est remplacé, son
		// ancrage de grâce (prev_token, prev_consumed_at) conservé. En Warn :
		// chaque passage ici est une réponse perdue côté client, un signal
		// réseau qu'il faut pouvoir compter en production.
		slog.WarnContext(ctx, "refresh accordé en fenêtre de grâce : rotation précédente jamais reçue par le client",
			"user", oldRefreshToken.UserID, "session", oldRefreshToken.Session,
			"version_remplacee", oldRefreshToken.TokenVersion.Int32,
			"version_emise", tokenPaire.RefreshTokenInfo.Version)
		err = qtx.GraceRotateRefreshToken(ctx, GraceRotateRefreshTokenParams{
			ID:           oldRefreshToken.ID,
			Token:        hashToken(tokenPaire.RefreshTokenInfo.Token),
			TokenVersion: services.ToPgInt4(tokenPaire.RefreshTokenInfo.Version),
			Expire:       services.ToPgTimestamptz(&tokenPaire.RefreshTokenInfo.Expiration),
			Created:      services.ToPgTimestamptz(&now),
		})
	} else {
		err = qtx.CreateRefreshToken(ctx, CreateRefreshTokenParams{
			Userid:       oldRefreshToken.UserID,
			Token:        hashToken(tokenPaire.RefreshTokenInfo.Token),
			Session:      tokenPaire.RefreshTokenInfo.Session,
			TokenVersion: services.ToPgInt4(tokenPaire.RefreshTokenInfo.Version),
			Expire:       services.ToPgTimestamptz(&tokenPaire.RefreshTokenInfo.Expiration),
			Created:      services.ToPgTimestamptz(&now),
			Revoked:      false,
			// Ancrage de la fenêtre de grâce : le jeton tout juste consommé
			// reste échangeable refreshGraceWindow durant (cf. jetonEnGrace).
			PrevToken:      services.ToPgText(hash),
			PrevConsumedAt: services.ToPgTimestamptz(&now),
		})
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, map[string]string{
		"accessToken":  tokenPaire.AccessToken.Token,
		"refreshToken": tokenPaire.RefreshTokenInfo.Token,
	})
}

type refreshBody struct {
	RefreshToken string `json:"refreshToken"`
}
