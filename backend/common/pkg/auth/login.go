package auth

import (
	"back-rex-common/pkg/services"
	"fmt"
	"net/http"
	"strconv"

	"context"
	"time"

	"github.com/go-chi/render"
)

type LoginRequest struct {
	Identifiant string `json:"identifiant"`
	Password    string `json:"password"`
}

type LoginResponse struct {
	Name         string `json:"name"`
	Surname      string `json:"surname"`
	Role         string `json:"role"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *LoginRequest) Bind(r *http.Request) error {
	if c.Identifiant == "" || c.Password == "" {
		return fmt.Errorf("identifiant et mot de passe requis")
	}
	return nil
}

func Login(w http.ResponseWriter, r *http.Request,
	jwtCfg services.JWTConfig, cfg services.LDAPConfig,
	postLdap LdapPostHandler) {

	data := &LoginRequest{}
	if err := render.Bind(r, data); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// 🔐 Authentification via LDAP
	ldapIdentity, err := LdapAuthenticate(cfg, data.Identifiant, data.Password)
	if err != nil {
		if validationErr, ok := err.(*services.AppValidationError); ok {
			services.InvalidRequestError(w, r, "Erreur validation", services.VALIDATION_ERROR, validationErr.Form)
		} else {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		}
		return
	}

	claim, subject, err := postLdap(r, ldapIdentity)
	if err != nil {
		if validationErr, ok := err.(*services.AppValidationError); ok {
			services.InvalidRequestError(w, r, "Erreur validation", services.VALIDATION_ERROR, validationErr.Form)
		} else {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		}
		return
	}
	// 🆕 Génération des tokens JWT

	tokenPaire, err := genereTokenPaire(jwtCfg, nil, claim, *subject)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())

	//sauvegarde le token dans la base de données
	queriesToken := New(pgCtx.Db)
	now := time.Now()

	userId, err := strconv.Atoi(*subject)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	err = queriesToken.CreateRefreshToken(context.Background(), CreateRefreshTokenParams{
		Userid:       int32(userId),
		Token:        tokenPaire.RefreshTokenInfo.Token,
		Expire:       services.ToPgTimestamptz(&tokenPaire.RefreshTokenInfo.Expiration),
		Session:      tokenPaire.RefreshTokenInfo.Session,
		TokenVersion: services.ToPgInt4(tokenPaire.RefreshTokenInfo.Version),
		Created:      services.ToPgTimestamptz(&now),
		Revoked:      false,
	})

	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	roles := (*claim)["roles"].(string)

	// ✅ Réponse avec le token
	render.JSON(w, r, &LoginResponse{
		Name:         data.Identifiant,
		Surname:      ldapIdentity.Surname,
		AccessToken:  tokenPaire.RefreshTokenInfo.Token,
		RefreshToken: tokenPaire.RefreshTokenInfo.Token,
		Role:         roles,
	})
}
