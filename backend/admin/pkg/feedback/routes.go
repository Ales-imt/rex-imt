package feedback

import (
	"github.com/go-chi/chi/v5"
)

func RouteFeedback(r chi.Router) {
	r.Get("/", GetAllFeedback)
}
