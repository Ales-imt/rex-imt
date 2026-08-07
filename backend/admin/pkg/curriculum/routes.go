package curriculum

import "github.com/go-chi/chi/v5"

func RouteCurriculum(r chi.Router) {
	r.Get("/annees", GetAnnees)
	r.Get("/annee-courante", GetAnneeCourante)
	r.Get("/promotions", GetPromotionTree)
}
