package annee

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func RouteAnnee(r chi.Router) {
	r.Post("/", CreateAnnee)
	r.Get("/", ListAnnee)
	r.Delete("/bulk", DeleteAnneeBulk)

	r.Route("/{anneeID}", func(r chi.Router) {
		r.Use(AnneeUse)
		r.Get("/", GetAnnee)
		r.Put("/", UpdateAnnee)
		r.Delete("/", DeleteAnnee)
	})
}

func AnneeUse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if anneeID := chi.URLParam(r, "anneeID"); anneeID != "" {
			pgCtx := services.GetPgCtx(r.Context())

			id, err := strconv.ParseInt(anneeID, 10, 64)
			if err != nil {
				services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

			queries := New(pgCtx.Db)
			annee, err := queries.GetAnneeById(context.Background(), id)
			if err == pgx.ErrNoRows {
				services.InvalidRequestError(w, r, "Année introuvable", services.NO_INFORMATION, nil)
				return
			}
			if err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}

			ctx := setAnneeFromCtx(r, &annee)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		services.InvalidRequestError(w, r, "pas d'id annee", services.NO_INFORMATION, nil)
	})
}
