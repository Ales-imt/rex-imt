package user

import (
	"back-rex-common/pkg/services"
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// RouteUtilisateur monte le CRUD utilisateur. mariaDb (Auréga) sert à dater les
// sorties lors d'une demande de suppression ; il peut être nil — l'admin reste
// alors utilisable, aucun compte n'étant anonymisé sans date de sortie connue.
func RouteUtilisateur(r chi.Router, cfg services.LDAPConfig, mariaDb *sql.DB) {
	aurega := NewAurega(mariaDb)

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		CreateUser(w, r, cfg)
	})

	r.Delete("/bulk", makeDeleteUserBulk(aurega))

	r.Route("/{userID}", func(r chi.Router) {
		r.Use(UserUse)      // Load the *Article on the request context
		r.Get("/", GetUser) // GET /articles/123
		r.Put("/", func(w http.ResponseWriter, r *http.Request) {
			UpdateUser(w, r, cfg)
		}) // PUT /articles/123
		r.Delete("/", makeDeleteUser(aurega)) // DELETE /articles/123
	})

	r.Get("/", ListUser)

}

func UserUse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if userID := chi.URLParam(r, "userID"); userID != "" {
			pgCtx := services.GetPgCtx(r.Context())

			id, err := strconv.Atoi(userID)
			if err != nil {
				services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

			queries := New(pgCtx.Db)
			user, err := queries.GetUserById(context.Background(), int32(id))
			if err == pgx.ErrNoRows {
				services.InvalidRequestError(w, r, "Utilisateur introuvable", services.NO_INFORMATION, nil)
				return
			}
			if err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

			ctx := setUserFromCtx(r, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		services.InvalidRequestError(w, r, "pas d'id utilisateur", services.NO_INFORMATION, nil)
	})
}
