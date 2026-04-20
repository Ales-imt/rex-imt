package auth

import (
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const pathAcessToken = "access_token"
const pathRefresh = "refresh_token"

type RefreshTokenInfo struct {
	Token      string
	Expiration time.Time
	Session    string
	Version    int
}
type TokenPair struct {
	AccessToken      *string    `json:"access_token"`
	ExpirationAccess *time.Time `json:"expiration_access"`
	RefreshTokenInfo *RefreshTokenInfo
}

func (t TokenPair) accessToCookies() http.Cookie {

	accessCokie := services.CreateCookie(pathAcessToken,
		"/", *t.ExpirationAccess, *t.AccessToken)

	return accessCokie

}

func (t TokenPair) refreshToCookies() http.Cookie {

	refreshCookie := services.CreateCookie(pathRefresh,
		"/", t.RefreshTokenInfo.Expiration, t.RefreshTokenInfo.Token)

	return refreshCookie

}

func genereTokenPaire(jwtCfg services.JWTConfig, oldRefreshToken *RefreshToken, atClaims *jwt.MapClaims, subject string) (*TokenPair, error) {
	refreshTokenInfo, err := generateRefreshToken(oldRefreshToken, subject, jwtCfg)
	if err != nil {
		return nil, err
	}

	// necessaire de connaitre la session dans access token pour le logout
	accessToken, expirationAccess, err := generateAccessToken(subject, jwtCfg, atClaims, refreshTokenInfo.Session)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		ExpirationAccess: expirationAccess,
		RefreshTokenInfo: refreshTokenInfo,
	}, nil
}

func generateAccessToken(subject string, jwtCfg services.JWTConfig, atClaims *jwt.MapClaims, session string) (*string, *time.Time, error) {

	expiration := time.Now().Add(jwtCfg.AccessTokenExpiresIn)
	claims := jwt.MapClaims{
		"sub":        subject,
		"exp":        expiration.Unix(),
		"session_id": session,
	}

	if atClaims != nil {
		for key, value := range *atClaims {
			claims[key] = value
		}
	}

	var err error

	AccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := AccessToken.SignedString([]byte(jwtCfg.Secret))
	if err != nil {
		return nil, nil, err
	}

	return &token, &expiration, nil

}

func generateRefreshToken(oldRefreshToken *RefreshToken, subject string, jwtCfg services.JWTConfig) (*RefreshTokenInfo, error) {

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = subject

	info := RefreshTokenInfo{}
	info.Expiration = time.Now().Add(jwtCfg.RefreshTokenExpiresIn)
	rtClaims["exp"] = info.Expiration.Unix()

	if oldRefreshToken == nil {
		info.Session = uuid.New().String()
		info.Version = 0
	} else {
		info.Session = oldRefreshToken.Session
		info.Version = int(oldRefreshToken.TokenVersion.Int32) + 1
	}

	rtClaims["session_id"] = info.Session
	rtClaims["version"] = info.Version

	var err error
	info.Token, err = refreshToken.SignedString([]byte(jwtCfg.Secret))
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func getClaims(r *http.Request, jwtSecret string, getToken func(r *http.Request) (string, error)) (*jwt.MapClaims, error) {

	access_token, err := getToken(r)
	if err != nil {
		return nil, err
	}

	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(access_token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	return &claims, nil
}

func getSubjectFromClaims(claims *jwt.MapClaims) (*string, error) {
	userId, ok := (*claims)["sub"].(string)
	if !ok {
		return nil, errors.New("sub claim missing or invalid")
	}

	return &userId, nil
}

func GetAccessTokenByBearer(r *http.Request) (string, error) {

	// sinon regarde dans les headers
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header missing")
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid auth header format")
	}

	return parts[1], nil

}

func GetAccessTokenByCookies(r *http.Request) (string, error) {

	// regarde si dans les cookie
	access_token, err := r.Cookie(pathAcessToken)

	if err != nil {
		return "", err
	}

	return access_token.Value, nil

}

func GetRefreshTokenByCookies(r *http.Request) (string, error) {

	// regarde si dans les cookie
	access_token, err := r.Cookie(pathRefresh)

	if err != nil {
		return "", err
	}

	return access_token.Value, nil

}

func StartRefreshTokenCleanup(cfg *services.DatabaseConfig) {
	dsn := services.ToDBS(cfg)
	pg := services.NewPG(context.Background(), dsn)

	go func() {
		queries := New(pg.Db)
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			// Supprime les tokens expirés
			err := queries.CleanUpTokens(context.Background())
			if err != nil {
				fmt.Printf("Erreur lors de la suppression des tokens expirés: %v", err)
			}
			<-ticker.C
		}
	}()
}

// Utile pour supprimer les cookies d'acces cote client sur un logout.
func genereInvalidTokenPaire() *TokenPair {

	noPayload := ""
	now := time.Now()

	pairs := TokenPair{
		AccessToken:      &noPayload,
		ExpirationAccess: &now,
		RefreshTokenInfo: &RefreshTokenInfo{
			Token:      noPayload,
			Expiration: now,
		},
	}

	return &pairs
}
