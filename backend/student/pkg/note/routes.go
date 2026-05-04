package note

import "github.com/go-chi/chi/v5"

func MakeRouteNote(connector NoteConnector) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", getNotes(connector))
	}
}
