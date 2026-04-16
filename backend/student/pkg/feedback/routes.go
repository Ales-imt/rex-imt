package feedback

import "github.com/go-chi/chi/v5"

func RouteFeedBack(r chi.Router) {
	r.Post("/", Feedback)
	r.Get("/last_elo_gain", GetLastEloGain)
}
