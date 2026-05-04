package programme

import "github.com/go-chi/chi/v5"

func MakeRouteProgramme(connector ProgrammeConnector) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", getProgramme(connector))
	}
}
