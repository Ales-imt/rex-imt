package authentification

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

func PostLdap(r *http.Request, ldapIdentity *auth.LdapIdentity) (*jwt.MapClaims, *string, error) {

	pgCtx := services.GetPgCtx(r.Context())
	queriesAuth := auth.New(pgCtx.Db)

	userByMail, err := queriesAuth.GetUserByMail(r.Context(), ldapIdentity.Mail)

	if err == pgx.ErrNoRows {
		return nil, nil, services.NewAppValidationError("Utilisateur inconnu", "identifiant")
	}

	if err != nil {
		return nil, nil, err
	}

	roles := ""
	if userByMail.Roles.Valid {
		roles = userByMail.Roles.String
	}

	claims := jwt.MapClaims{"roles": roles}
	subject := strconv.Itoa(int(userByMail.ID))
	return &claims, &subject, nil // Pas de claims supplémentaires pour l'instant
}
