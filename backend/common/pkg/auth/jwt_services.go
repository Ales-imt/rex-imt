package auth

import (
	"back-rex-common/pkg/services"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

const pathAcessToken = "access_token"
const pathRefresh = "refresh_token"

type RefreshTokenInfo struct {
	Token      string
	Expiration time.Time
	Session    string
	Version    int
}

type AccessTokenInfo struct {
	Token      string
	Expiration time.Time
}

type TokenPair struct {
	AccessToken      *AccessTokenInfo
	RefreshTokenInfo *RefreshTokenInfo
}

func genereTokenPaire(jwtCfg services.JWTConfig, oldRefreshToken *RefreshToken, atClaims *jwt.MapClaims, subject string) (*TokenPair, error) {
	refreshTokenInfo, err := generateRefreshToken(oldRefreshToken, subject, jwtCfg)
	if err != nil {
		return nil, err
	}

	// necessaire de connaitre la session dans access token pour le logout
	accessToken, err := generateAccessToken(subject, jwtCfg, atClaims, refreshTokenInfo.Session)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshTokenInfo: refreshTokenInfo,
	}, nil
}

func generateAccessToken(subject string, jwtCfg services.JWTConfig, atClaims *jwt.MapClaims, session string) (*AccessTokenInfo, error) {

	now := time.Now()
	expiration := now.Add(jwtCfg.AccessTokenExpiresIn)
	claims := jwt.MapClaims{
		"sub":        subject,
		"exp":        expiration.Unix(),
		"iat":        now.Unix(),
		"nbf":        now.Unix(),
		"jti":        uuid.New().String(),
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
		return nil, err
	}

	return &AccessTokenInfo{
		Token:      token,
		Expiration: expiration,
	}, nil

}

func generateRefreshToken(oldRefreshToken *RefreshToken, subject string, jwtCfg services.JWTConfig) (*RefreshTokenInfo, error) {

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = subject

	info := RefreshTokenInfo{}
	info.Expiration = time.Now().Add(jwtCfg.RefreshTokenExpiresIn)
	rtClaims["exp"] = info.Expiration.Unix()

	info.Session = uuid.New().String()
	if oldRefreshToken == nil {
		info.Version = 0
	} else {
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

func getClaims(r *http.Request, jwtSecret string) (*jwt.MapClaims, error) {

	access_token, err := GetAccessTokenByBearer(r)
	if err != nil {
		return nil, err
	}

	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(access_token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
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
