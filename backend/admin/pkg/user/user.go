package user

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	usercommon "back-rex-common/pkg/user"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/go-ldap/ldap/v3"
	"github.com/jackc/pgx/v5"
)

type UserRequest struct {
	ID      int32  `json:"id"`
	Version int    `json:"version"`
	Email   string `json:"email"`
	Roles   string `json:"roles"`
	Blame   bool   `json:"blame"`
}

func CreateUser(w http.ResponseWriter, r *http.Request, cfg services.LDAPConfig) {
	var input UserRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	sr, err := getLdapUser(input.Email, cfg)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	issues := validateUser(input, sr)
	if len(issues) != 0 {
		services.InvalidRequestError(w, r, "Invalid user", services.VALIDATION_ERROR, issues)
		return
	}

	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)
	tx, err := pgCtx.Db.Begin(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	defer tx.Rollback(ctx)

	queries := New(tx)
	_, err = queries.GetUserByMail(ctx, input.Email)

	if err == nil {
		issues := []services.FormValidation{{Path: "email", Message: "Utilisateur existant"}}
		services.InvalidRequestError(w, r, "Invalid user", services.VALIDATION_ERROR, issues)
		return
	} else if err != pgx.ErrNoRows {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	etudiant := len(input.Roles) != 0 && strings.Contains(input.Roles, "etudiant")
	ldapIdentity := auth.GetLdapIdentity(sr.Entries[0])

	id, err := usercommon.CreateUser(tx, ldapIdentity, ctx, input.Roles, etudiant)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	tx.Commit(r.Context())
	input.ID = int32(id)

	render.JSON(w, r, input)
}

var allowedRoles = map[string]struct{}{
	"admin":        {},
	"etudiant":     {},
	"gestionnaire": {},
}

func validateUser(user UserRequest, sr *ldap.SearchResult) []services.FormValidation {
	issues := []services.FormValidation{}

	if len(user.Roles) == 0 {
		issues = append(issues, services.FormValidation{
			Path:    "roles",
			Message: "Un role doit etre précisé",
		})
	} else {
		for _, role := range strings.Split(user.Roles, ",") {
			role = strings.TrimSpace(role)
			if _, ok := allowedRoles[role]; !ok {
				issues = append(issues, services.FormValidation{
					Path:    "roles",
					Message: fmt.Sprintf("role %s non autorisé", role),
				})
			}
		}
	}

	if user.Email == "" {
		issues = append(issues, services.FormValidation{
			Path:    "email",
			Message: "Email obligatoire",
		})
	} else {

		if len(sr.Entries) == 0 {
			issues = append(issues, services.FormValidation{
				Path:    "email",
				Message: "ldap user not found",
			})
		}
	}

	return issues
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r)
	render.JSON(w, r, user)
}

func ListUser(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	pgctx := services.GetPgCtx(ctx)
	queries := New(pgctx.Db)

	users, err := queries.ListUser(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if users == nil {
		users = []User{}
	}

	render.JSON(w, r, users)
}

func UpdateUser(w http.ResponseWriter, r *http.Request, cfg services.LDAPConfig) {

	var input UserRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	oldUser := getUserFromCtx(r)

	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)
	tx, err := pgCtx.Db.Begin(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	defer tx.Rollback(ctx)
	queries := New(tx)

	user, err := queries.UpdatePartialUser(ctx, UpdatePartialUserParams{
		ID:      oldUser.ID,
		Version: oldUser.Version,
		Roles:   services.ToPgText(input.Roles),
		Blame:   services.ToPgBool(input.Blame),
	})

	if err == pgx.ErrNoRows {
		services.ConflictError(w, r, "", services.OPTIMISTIC_LOCKING_FAILURE, nil)
		return
	}

	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	etudiant := len(input.Roles) != 0 && strings.Contains(input.Roles, "etudiant")

	if etudiant {
		queriesCommon := usercommon.New(tx)
		exist, err := queriesCommon.IsStudentExist(ctx, user.ID)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		if !exist {
			// essai de recuperer la promotion dans ldap
			sr, err := getLdapUser(user.Email, cfg)
			if err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

			ldapIdentity := auth.GetLdapIdentity(sr.Entries[0])

			err = queriesCommon.CreateStudent(ctx, usercommon.CreateStudentParams{
				UserID:    user.ID,
				Promotion: services.ToPgText(ldapIdentity.Promotion),
			})
			if err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

		}

	}

	tx.Commit(ctx)

	render.JSON(w, r, user)

}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r)

	ctx := r.Context()
	pgctx := services.GetPgCtx(ctx)
	queries := New(pgctx.Db)

	err := queries.DeleteUser(ctx, user.ID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return

	}

	w.WriteHeader(http.StatusNoContent)
}

type MailCheck struct {
	Exist bool `json:"exist"`
}

func getLdapUser(email string, cfg services.LDAPConfig) (*ldap.SearchResult, error) {
	// Connexion au serveur LDAP
	l, err := ldap.DialURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("LDAP connection failed: %w", err)
	}
	defer l.Close()

	filter := fmt.Sprintf("(mail=%s)", ldap.EscapeFilter(email))

	searchRequest := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		filter,
		[]string{"*"}, // si nil, retourne ts les attibuts.
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	return sr, nil
}
