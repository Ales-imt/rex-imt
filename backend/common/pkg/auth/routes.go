package auth

import (
	"back-rex-common/pkg/mailer"
	"back-rex-common/pkg/services"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RoutesAuth(r chi.Router, cfg *services.Config,
	postLdap LdapPostHandler) {
	r.With(RateLimitLogin).Post("/login", func(w http.ResponseWriter, r *http.Request) {
		Login(w, r, cfg.JWT, cfg.LDAP, postLdap)
	})
	r.With(RateLimitLogin).Post("/email/request", func(w http.ResponseWriter, r *http.Request) {
		RequestEmailCode(w, r, cfg.SMTP, mailer.Send)
	})
	r.With(RateLimitLogin).Post("/email/verify", func(w http.ResponseWriter, r *http.Request) {
		VerifyEmailCode(w, r, cfg.JWT)
	})
	r.With(Security(cfg.JWT, nil)).Get("/logout",
		func(w http.ResponseWriter, r *http.Request) {
			Logout(w, r, cfg.JWT)
		})

	r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
		RefreshAccessToken(w, r, cfg.JWT)
	})

}
