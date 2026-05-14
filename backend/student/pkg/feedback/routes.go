package feedback

import "github.com/go-chi/chi/v5"

func MakeRouteFeedBack(agePublicKey string) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", makeFeedbackHandler(agePublicKey))
	}
}
