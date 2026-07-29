package authentification

import (
	"back-rex-common/pkg/auth"

	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

func PostLdap(r *http.Request, ldapIdentity *auth.LdapIdentity) (*jwt.MapClaims, *string, error) {
	pgCtx := services.GetPgCtx(r.Context())
	queriesAuth := auth.New(pgCtx.Db)
	return postLdap(r.Context(), queriesAuth, ldapIdentity)
}

// postLdap contient la logique de résolution (utilisateur connu/inconnu,
// rôles), indépendamment du contexte HTTP : testable avec un DBTX en mémoire
// (voir post_ldap_test.go).
func postLdap(ctx context.Context, queriesAuth *auth.Queries, ldapIdentity *auth.LdapIdentity) (*jwt.MapClaims, *string, error) {
	userByMail, err := queriesAuth.GetUserByMail(ctx, ldapIdentity.Mail)
	if err == pgx.ErrNoRows {
		return nil, nil, services.NewAppValidationError("Utilisateur inconnu", "identifiant")
	}
	if err != nil {
		return nil, nil, err
	}

	roles := auth.RoleEleve
	if len(userByMail.Roles) > 0 {
		roles = strings.Join(userByMail.Roles, ",")
	}
	claims := jwt.MapClaims{"roles": roles}
	subject := strconv.Itoa(int(userByMail.ID))
	return &claims, &subject, nil
}
