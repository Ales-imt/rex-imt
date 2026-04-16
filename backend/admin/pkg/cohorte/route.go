package cohorte

import (
	"back-rex-common/pkg/services"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RouteCohorte(r chi.Router, ldapConfig services.LDAPConfig) {
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		ImportCohorte(w, r, ldapConfig)
	})

	r.Get("/promo", GetPromo)
	r.Get("/groupe", GetGroupe)
}
