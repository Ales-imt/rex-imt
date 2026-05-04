package programme

import "github.com/go-chi/chi/v5"

func MakeRouteProgramme() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", GetProgramme)
	}
}
