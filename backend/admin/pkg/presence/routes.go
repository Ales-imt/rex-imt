package presence

import "github.com/go-chi/chi/v5"

func RoutePresence(r chi.Router, secret string, planningURL string) {
	SetTokenSecret(secret)

	r.Get("/matieres/{matiereId}/planning", GetPlanningHandler(planningURL))
	r.Post("/seance", OpenSeanceHandler)
	r.Post("/seance/{seanceId}/close", CloseSeanceHandler)
	r.Get("/seance/{seanceId}/token", GetTokenHandler)
	r.Get("/seance/{seanceId}/presence", GetPresenceHandler)
	r.Get("/matieres/{matiereId}/seances", ListSeancesHandler)
}
