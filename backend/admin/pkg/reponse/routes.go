package reponse

import "github.com/go-chi/chi/v5"

func RouteReponse(r chi.Router) {
	r.Post("/", PostReponse)
	r.Get("/{feedbackId}", GetReponses)
}
