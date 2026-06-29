package presence

import "github.com/go-chi/chi/v5"

func RoutePresence(r chi.Router, secret string) {
	SetTokenSecret(secret)

	r.Get("/matieres/{matiereId}/planning", GetPlanningHandler)
	r.Post("/seance", OpenSeanceHandler)
	r.Post("/seance/{seanceId}/close", CloseSeanceHandler)
	r.Get("/seance/{seanceId}/token", GetTokenHandler)
	r.Get("/seance/{seanceId}/presence", GetPresenceHandler)
	r.Get("/matieres/{matiereId}/seances", ListSeancesHandler)
	r.Get("/matieres/{matiereId}/seance/slot", GetSlotHandler)
}
