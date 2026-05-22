package postit

import "github.com/go-chi/chi/v5"

func MakeRoutePostit() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", GetPostits)
		r.Get("/promotions", GetPromotions)
	}
}
