package evaluation

import (
	"back-rex-eleve/pkg/programme"

	"github.com/go-chi/chi/v5"
)

func MakeRouteEvaluation(connector programme.ProgrammeConnector, agePublicKey string) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/", SubmitEvaluation(agePublicKey))
		r.Get("/matieres", getMatiere(connector))
	}
}
