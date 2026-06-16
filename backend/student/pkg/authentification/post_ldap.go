package authentification

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/user"

	"back-rex-common/pkg/services"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

func PostLdap(r *http.Request, ldapIdentity *auth.LdapIdentity) (*jwt.MapClaims, *string, error) {

	// ✅ Vérifie si l'étudiant existe déjà dans la base

	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)

	tx, err := pgCtx.Db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	queriesAuth := auth.New(pgCtx.Db)
	userByMail, err := queriesAuth.GetUserByMail(ctx, ldapIdentity.Mail)

	var id int

	switch err {
	case nil:
		id = int(userByMail.ID)
	case pgx.ErrNoRows: // 🆕 S'il n'existe pas, on l’insère dans la table `student`
		id, err = user.CreateUser(tx, ldapIdentity, ctx, []string{auth.RoleEleve}, true)
		if err != nil {
			return nil, nil, err
		}
		err = tx.Commit(ctx)
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, err
	}

	claims := jwt.MapClaims{"roles": auth.RoleEleve}
	subject := strconv.Itoa(id)

	return &claims, &subject, nil // Pas de claims supplémentaires pour l'instant
}
