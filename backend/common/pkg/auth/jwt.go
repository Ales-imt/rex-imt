package auth

import (
	"back-rex-common/pkg/services"
	"strconv"

	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

func RefreshAccessToken(w http.ResponseWriter, r *http.Request, jwtConfig services.JWTConfig) {

	pgCtx := services.GetPgCtx(r.Context())

	oldRefreshToken, err := checkRefreshToken(r, pgCtx)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// Get the user
	queries := New(pgCtx.Db)
	user, err := queries.GetUserById(context.Background(), oldRefreshToken.UserID)
	if err == pgx.ErrNoRows {
		services.InvalidRequestError(w, r, "Utilisateur inconnu", services.NO_INFORMATION, nil)
		return
	}

	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if user.Blame.Valid && user.Blame.Bool {
		services.AuthorizationError(w, r, "Utilisateur banni", services.NO_INFORMATION, nil)
		return
	}

	claims := jwt.MapClaims{"roles": user.Roles}

	tokenPaire, err := genereTokenPaire(jwtConfig, oldRefreshToken, &claims, strconv.Itoa(int(oldRefreshToken.UserID)))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	accessCokie := tokenPaire.accessToCookies()
	http.SetCookie(w, &accessCokie)

	refreshCookie := tokenPaire.refreshToCookies()
	http.SetCookie(w, &refreshCookie)

	err = queries.DeleteRefreshToken(context.Background(), oldRefreshToken.Token)

	if err == pgx.ErrNoRows {
		services.InvalidRequestError(w, r, "Refresh token inconnu", services.NO_INFORMATION, nil)
		return
	}

	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	now := time.Now()
	err = queries.CreateRefreshToken(context.Background(), CreateRefreshTokenParams{
		Userid:       oldRefreshToken.UserID,
		Token:        tokenPaire.RefreshTokenInfo.Token,
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

	w.WriteHeader(http.StatusNoContent) // 204 No Content

}

func Me(w http.ResponseWriter, r *http.Request, jwtConfig services.JWTConfig) {

	_, err := checkRefreshToken(r, services.GetPgCtx(r.Context()))
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	claim, err := getClaims(r, jwtConfig.Secret, GetRefreshTokenByCookies)
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
		Role:    user.Roles.String,
	})

}

func checkRefreshToken(r *http.Request, pgCtx *services.Postgres) (*RefreshToken, error) {

	refreshToken, err := r.Cookie(pathRefresh)
	if err != nil {
		return nil, err
	}

	queries := New(pgCtx.Db)
	// Retrieve the refresh token
	token, err := queries.GetRefreshToken(context.Background(), refreshToken.Value)
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
