package authentification

// Vérifie que l'auto-inscription a bien été supprimée (chantier 1) : un
// utilisateur LDAP absent de la table `user` est refusé, aucune ligne n'est
// créée. Un DBTX en mémoire remplace la base réelle (même principe que
// witnessDB dans backend/admin/pkg/presence/witness_test.go).

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeUserRow struct {
	id    int32
	roles []string
}

// fakeUserDB implémente auth.DBTX pour GetUserByMail uniquement : si
// postLdap tentait d'écrire (INSERT/UPDATE), Exec renverrait une erreur et le
// test échouerait — c'est la garantie que rien n'est créé pour un inconnu.
type fakeUserDB struct {
	usersByMail map[string]fakeUserRow
}

func (f *fakeUserDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("aucune écriture attendue, reçu Exec: " + sql)
}

func (f *fakeUserDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query: " + sql)
}

type fakeRow struct {
	err  error
	scan func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return r.scan(dest...)
}

func (f *fakeUserDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	email := args[0].(string)
	u, ok := f.usersByMail[email]
	if !ok {
		return &fakeRow{err: pgx.ErrNoRows}
	}
	return &fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*int32)) = u.id
		*(dest[1].(*pgtype.Int4)) = pgtype.Int4{Int32: 1, Valid: true}
		*(dest[2].(*string)) = "Nom"
		*(dest[3].(*string)) = "Prenom"
		*(dest[4].(*string)) = email
		*(dest[5].(*[]string)) = u.roles
		*(dest[6].(*pgtype.Bool)) = pgtype.Bool{}
		return nil
	}}
}

func TestPostLdap_UnknownUser_Refused(t *testing.T) {
	db := &fakeUserDB{usersByMail: map[string]fakeUserRow{}}
	queries := auth.New(db)

	identity := &auth.LdapIdentity{Name: "Trens", Surname: "Clement", Mail: "inconnu@mines-ales.fr"}

	claims, subject, err := postLdap(context.Background(), queries, identity)

	if err == nil {
		t.Fatal("un utilisateur LDAP absent de la table user doit être refusé")
	}
	validationErr, ok := err.(*services.AppValidationError)
	if !ok {
		t.Fatalf("attendu une AppValidationError, reçu %T: %v", err, err)
	}
	if validationErr.Form.Message != "Utilisateur inconnu" {
		t.Fatalf("message attendu \"Utilisateur inconnu\", reçu %q", validationErr.Form.Message)
	}
	if claims != nil || subject != nil {
		t.Fatal("aucun claim ni subject ne doit être renvoyé pour un inconnu")
	}
}

func TestPostLdap_KnownUser_RolesFromDB(t *testing.T) {
	db := &fakeUserDB{usersByMail: map[string]fakeUserRow{
		"prof@mines-ales.fr": {id: 7, roles: []string{auth.RoleProf}},
	}}
	queries := auth.New(db)

	identity := &auth.LdapIdentity{Name: "Dupont", Surname: "Jean", Mail: "prof@mines-ales.fr"}

	claims, subject, err := postLdap(context.Background(), queries, identity)
	if err != nil {
		t.Fatalf("un utilisateur connu doit être accepté: %v", err)
	}
	if subject == nil || *subject != "7" {
		t.Fatalf("subject attendu \"7\", reçu %v", subject)
	}
	if (*claims)["roles"] != auth.RoleProf {
		t.Fatalf("roles attendu %q, reçu %v", auth.RoleProf, (*claims)["roles"])
	}
}

func TestPostLdap_KnownUser_EmptyRolesFallbackToEleve(t *testing.T) {
	db := &fakeUserDB{usersByMail: map[string]fakeUserRow{
		"eleve@mines-ales.fr": {id: 3, roles: nil},
	}}
	queries := auth.New(db)

	identity := &auth.LdapIdentity{Name: "Martin", Surname: "Alice", Mail: "eleve@mines-ales.fr"}

	claims, _, err := postLdap(context.Background(), queries, identity)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if (*claims)["roles"] != auth.RoleEleve {
		t.Fatalf("repli attendu %q, reçu %v", auth.RoleEleve, (*claims)["roles"])
	}
}
