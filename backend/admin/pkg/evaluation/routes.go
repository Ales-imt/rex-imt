package evaluation

import (
	"back-rex-admin/pkg/ia"

	"github.com/go-chi/chi/v5"
)

func RouteEvaluation(r chi.Router, connector ia.IAConnector) {
	r.Get("/annees", GetAnnees)
	r.Get("/promotions", GetPromotionTree)
	r.Get("/matieres/{matiereId}/stats", GetMatiereStats)
	r.Get("/matieres/{matiereId}/verbatims", GetVerbatims)
	r.Post("/matieres/{matiereId}/synthese", GenerateSynthese(connector))
	r.Patch("/syntheses/{syntheseId}/statut", UpdateSyntheseStatut)
}
