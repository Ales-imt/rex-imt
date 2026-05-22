package reponse

import "github.com/go-chi/chi/v5"

func MakeRouteReponse() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", GetUnreadReponses)
		r.Get("/history", GetChatHistory)
	}
}
