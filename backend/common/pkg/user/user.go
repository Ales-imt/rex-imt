package user

import (
	"back-rex-common/pkg/auth"
	"context"

	"github.com/jackc/pgx/v5"
)

func CreateUser(tx pgx.Tx, ldapIdentity *auth.LdapIdentity, ctx context.Context, roles []string, student bool) (int, error) {
	queries := New(tx)
	newId, err := queries.CreateUser(ctx, CreateUserParams{
		Name:    ldapIdentity.Name,
		Surname: ldapIdentity.Surname,
		Email:   ldapIdentity.Mail,
		Roles:   roles,
	})
	if err != nil {
		return -1, err
	}

	if student {
		err = queries.CreateStudent(ctx, newId)
		if err != nil {
			return -1, err
		}
	}

	return int(newId), nil
}
