package auth

import (
	"back-rex-common/pkg/services"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RoutesAuth(r chi.Router, cfg *services.Config,
	postLdap LdapPostHandler) {
	r.With(RateLimitLogin).Post("/login", func(w http.ResponseWriter, r *http.Request) {
		Login(w, r, cfg.JWT, cfg.LDAP, postLdap)
	})
	r.With(Security(cfg.JWT, nil)).Get("/logout",
		func(w http.ResponseWriter, r *http.Request) {
			Logout(w, r, cfg.JWT)
		})

	r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
		RefreshAccessToken(w, r, cfg.JWT)
	})

	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		Me(w, r, cfg.JWT)
	})

}
