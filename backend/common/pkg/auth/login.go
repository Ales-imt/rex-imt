package auth

import (
	"back-rex-common/pkg/services"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/render"
)

type LoginRequest struct {
	Identifiant string `json:"identifiant"`
	Password    string `json:"password"`
}

type LoginResponse struct {
	Name         string   `json:"name"`
	Surname      string   `json:"surname"`
	Roles        []string `json:"roles"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
}

func (c *LoginRequest) Bind(r *http.Request) error {
	if c.Identifiant == "" || c.Password == "" {
		return fmt.Errorf("identifiant et mot de passe requis")
	}
	if len(c.Identifiant) > 254 || len(c.Password) > 256 {
		return fmt.Errorf("identifiant ou mot de passe trop long")
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
			services.InternalServerError(w, r, "Erreur d'authentification", services.NO_INFORMATION, nil)
		}
		return
	}

	claim, subject, err := postLdap(r, ldapIdentity)
	if err != nil {
		if validationErr, ok := err.(*services.AppValidationError); ok {
			services.InvalidRequestError(w, r, "Erreur validation", services.VALIDATION_ERROR, validationErr.Form)
		} else {
			services.InternalServerError(w, r, "Erreur interne", services.NO_INFORMATION, nil)
		}
		return
	}

	userId, err := strconv.Atoi(*subject)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	rolesStr, _ := (*claim)["roles"].(string)
	roles := strings.Split(rolesStr, ",")

	// 🆕 Génération des tokens JWT + réponse
	if err := issueSession(w, r, jwtCfg, userId, roles); err != nil {
		if errors.Is(err, ErrUserBlamed) {
			services.AuthorizationError(w, r, "Utilisateur banni", services.NO_INFORMATION, nil)
			return
		}
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
}
