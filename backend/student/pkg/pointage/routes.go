package pointage

import "github.com/go-chi/chi/v5"

func MakeRoutePointage() func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", PostPointage())
	}
}
