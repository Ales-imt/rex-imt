package user

import "github.com/go-chi/chi/v5"

func MakeRouteUser() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", getMeHandler)
		r.Patch("/informed", setInformedAtHandler)
	}
}
