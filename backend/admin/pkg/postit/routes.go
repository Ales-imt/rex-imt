package postit

import "github.com/go-chi/chi/v5"

func RoutePostit(r chi.Router) {
	r.Post("/", PostPostit)
	r.Get("/", GetPostits)
}
