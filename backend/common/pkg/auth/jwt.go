package auth

import (
	"back-rex-common/pkg/services"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"context"
	"errors"
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

// RefreshAccessToken tourne le refresh token (rotation stricte, usage unique).
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

	oldRefreshToken, err := qtx.ConsumeRefreshToken(ctx, hashToken(body.RefreshToken))
	if err == pgx.ErrNoRows {
		// Jeton inconnu, déjà tourné, ou perdu face à une requête concurrente :
		// le log qualifie, le client reçoit un 400 volontairement opaque.
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
	err = qtx.CreateRefreshToken(ctx, CreateRefreshTokenParams{
		Userid:       oldRefreshToken.UserID,
		Token:        hashToken(tokenPaire.RefreshTokenInfo.Token),
		Session:      tokenPaire.RefreshTokenInfo.Session,
		TokenVersion: services.ToPgInt4(tokenPaire.RefreshTokenInfo.Version),
		Expire:       services.ToPgTimestamptz(&tokenPaire.RefreshTokenInfo.Expiration),
		Created:      services.ToPgTimestamptz(&now),
		Revoked:      false,
	})
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

func Me(w http.ResponseWriter, r *http.Request, jwtConfig services.JWTConfig) {

	_, err := checkRefreshToken(r, services.GetPgCtx(r.Context()))
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	claim, err := getClaims(r, jwtConfig.Secret)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	subject, err := getSubjectFromClaims(claim)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	id, err := strconv.Atoi(*subject)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())

	studentQueries := New(pgCtx.Db)
	user, err := studentQueries.GetUserById(context.Background(), int32(id))
	if err == pgx.ErrNoRows {
		services.InvalidRequestError(w, r, "Utilisateur inconnu", services.NO_INFORMATION, nil)
		return
	}

	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, &LoginResponse{
		Name:    user.Name,
		Surname: user.Surname,
		Roles:   user.Roles,
	})

}

type refreshBody struct {
	RefreshToken string `json:"refreshToken"`
}

func checkRefreshToken(r *http.Request, pgCtx *services.Postgres) (*RefreshToken, error) {

	var body refreshBody
	if err := render.DecodeJSON(r.Body, &body); err != nil || body.RefreshToken == "" {
		return nil, errors.New("refresh token manquant")
	}
	tokenValue := hashToken(body.RefreshToken)

	queries := New(pgCtx.Db)
	// Retrieve the refresh token
	token, err := queries.GetRefreshToken(context.Background(), tokenValue)
	if err != nil {
		return nil, err
	}

	// Check if the token is valid
	if token.Revoked {
		return nil, errors.New("token has been revoked")
	}

	// Check if the token has expired
	now := time.Now()
	if now.After(token.ExpiresAt.Time) {
		return nil, errors.New("token has been expires")
	}

	return &token, nil
}
