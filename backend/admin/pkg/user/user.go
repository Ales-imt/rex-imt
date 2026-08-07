package user

import (
	"back-rex-admin/pkg/account"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	usercommon "back-rex-common/pkg/user"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/go-ldap/ldap/v3"
	"github.com/jackc/pgx/v5"
)

type UserRequest struct {
	ID         int32    `json:"id"`
	Version    int      `json:"version"`
	Name       string   `json:"name"`
	Surname    string   `json:"surname"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
	Blame      bool     `json:"blame"`
	AuthSource string   `json:"auth_source"`
}

// CreateUser provisionne un utilisateur. La source d'authentification est
// déduite du domaine de l'e-mail (voir auth.AuthSourceForEmail) :
//   - "ldap" : domaine interne (@mines-ales.fr / @etu.mines-ales.fr), compte
//     interne dont l'identité (nom/prénom) est vérifiée contre l'annuaire LDAP.
//   - "email" : tout autre domaine, intervenant externe sans mot de passe ni
//     entrée LDAP, authentifié plus tard par code à usage unique (voir
//     back-rex-common/pkg/auth/email_otp.go). Rôle PROF obligatoire.
func CreateUser(w http.ResponseWriter, r *http.Request, cfg services.LDAPConfig) {
	var input UserRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	authSource := auth.AuthSourceForEmail(input.Email)

	var ldapIdentity *auth.LdapIdentity
	if authSource == auth.AuthSourceLDAP {
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
		ldapIdentity = auth.GetLdapIdentity(sr.Entries[0])
	} else {
		issues := validateExternalUser(input)
		if len(issues) != 0 {
			services.InvalidRequestError(w, r, "Invalid user", services.VALIDATION_ERROR, issues)
			return
		}
		ldapIdentity = &auth.LdapIdentity{
			Name:    input.Name,
			Surname: input.Surname,
			Mail:    strings.ToLower(input.Email),
		}
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

	etudiant := slices.Contains(input.Roles, auth.RoleEleve)

	id, err := usercommon.CreateUser(tx, ldapIdentity, ctx, input.Roles, etudiant, authSource)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	tx.Commit(r.Context())
	input.ID = int32(id)
	input.AuthSource = authSource

	render.JSON(w, r, input)
}

// validateExternalUser valide un intervenant externe (auth_source="email") :
// pas de vérification LDAP, mais rôle PROF obligatoire (ce sont les seuls
// comptes "email" prévus, voir docs chantier auth).
func validateExternalUser(user UserRequest) []services.FormValidation {
	issues := []services.FormValidation{}

	if len(user.Roles) == 0 {
		issues = append(issues, services.FormValidation{
			Path:    "roles",
			Message: "Un role doit etre précisé",
		})
	} else {
		for _, role := range user.Roles {
			role = strings.TrimSpace(role)
			if _, ok := allowedRoles[role]; !ok {
				issues = append(issues, services.FormValidation{
					Path:    "roles",
					Message: fmt.Sprintf("role %s non autorisé", role),
				})
			}
		}
		if !slices.Contains(user.Roles, auth.RoleProf) {
			issues = append(issues, services.FormValidation{
				Path:    "roles",
				Message: "un intervenant externe doit avoir le rôle PROF",
			})
		}
	}

	if user.Name == "" {
		issues = append(issues, services.FormValidation{Path: "name", Message: "Nom obligatoire"})
	}
	if user.Surname == "" {
		issues = append(issues, services.FormValidation{Path: "surname", Message: "Prénom obligatoire"})
	}
	if user.Email == "" || !services.IsValidEmail(user.Email) {
		issues = append(issues, services.FormValidation{Path: "email", Message: "Email invalide"})
	}

	return issues
}

var allowedRoles = map[string]struct{}{
	auth.RoleAdmin:        {},
	auth.RoleEleve:        {},
	auth.RoleGestionnaire: {},
	auth.RoleProf:         {},
	auth.RoleModerateur:   {},
}

func validateUser(user UserRequest, sr *ldap.SearchResult) []services.FormValidation {
	issues := []services.FormValidation{}

	if len(user.Roles) == 0 {
		issues = append(issues, services.FormValidation{
			Path:    "roles",
			Message: "Un role doit etre précisé",
		})
	} else {
		for _, role := range user.Roles {
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
		Name:    input.Name,
		Surname: input.Surname,
		Email:   input.Email,
		Roles:   input.Roles,
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

	// Bannir un utilisateur doit couper ses sessions immédiatement : on
	// supprime tous ses refresh tokens dans la même transaction. Sans cela le
	// ban n'agirait qu'au prochain rafraîchissement (contrôle jwt.go).
	if input.Blame {
		if err := auth.New(tx).DeleteRefreshTokensByUser(ctx, user.ID); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
	}

	etudiant := slices.Contains(input.Roles, auth.RoleEleve)

	if etudiant {
		queriesCommon := usercommon.New(tx)
		exist, err := queriesCommon.IsStudentExist(ctx, user.ID)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		if !exist {
			err = queriesCommon.CreateStudent(ctx, user.ID)
			if err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}
		}

	}

	tx.Commit(ctx)

	render.JSON(w, r, user)

}

// makeDeleteUser supprime, anonymise ou conserve un compte selon qu'il porte ou
// non des données de présence soumises à conservation. La décision est prise par
// account.Resolve, partagée avec la purge RGPD automatique.
//
// La réponse dit ce qui a réellement été fait : on n'annonce jamais une
// suppression qui n'a pas eu lieu.
func makeDeleteUser(aurega *auregaSortie) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromCtx(r)

		ctx := r.Context()
		pool := services.GetPgCtx(ctx).Db

		decision, err := account.Resolve(ctx, pool, pool, user.ID,
			aurega.lookup(), time.Now(), comptesAnonymizeAfterYears)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		if decision.Outcome == account.OutcomeDeleted {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		render.JSON(w, r, decision)
	}
}

type deleteBulkRequest struct {
	Ids []int32 `json:"ids"`
}

// makeDeleteUserBulk applique la même règle, compte par compte.
//
// Volontairement PAS de DELETE ... WHERE id = ANY(...) : un seul identifiant
// porteur de présence ferait échouer tout le lot. Chaque compte est traité
// indépendamment et le récapitulatif indique le sort de chacun.
func makeDeleteUserBulk(aurega *auregaSortie) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input deleteBulkRequest
		if err := render.DecodeJSON(r.Body, &input); err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		ctx := r.Context()
		pool := services.GetPgCtx(ctx).Db
		lookup := aurega.lookup()
		now := time.Now()

		results := make([]account.Decision, 0, len(input.Ids))
		for _, id := range input.Ids {
			decision, err := account.Resolve(ctx, pool, pool, id, lookup, now, comptesAnonymizeAfterYears)
			if err != nil {
				results = append(results, account.Decision{
					UserID:  id,
					Outcome: "erreur",
					Message: err.Error(),
				})
				continue
			}
			results = append(results, decision)
		}

		render.JSON(w, r, results)
	}
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
