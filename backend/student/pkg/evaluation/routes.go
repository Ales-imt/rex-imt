package evaluation

import (
	"github.com/go-chi/chi/v5"
)

func MakeRouteEvaluation(agePublicKey string) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", SubmitEvaluation(agePublicKey))
		r.Get("/matieres", getMatiere())
		r.Get("/session", getSessionDetail())
		r.Delete("/session/{sessionId}", deleteSession())
	}
}
