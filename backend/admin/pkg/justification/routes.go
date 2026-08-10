package justification

import "github.com/go-chi/chi/v5"

// RouteJustification monte les routes d'excuse. Le contrôle de rôle
// (ADMIN + GESTIONNAIRE) est appliqué au montage, dans cmd/main.go, comme pour
// les autres domaines du service admin.
//
// Ces routes n'existent QUE dans back-rex-admin : aucun chemin de création
// d'excuse n'est exposé au service étudiant.
func RouteJustification(r chi.Router) {
	r.Get("/preview", PreviewHandler)
	r.Get("/", ListHandler)
	r.Post("/", CreateHandler)
	r.Get("/{id}/seances", SeancesHandler)
	r.Put("/{id}", UpdateHandler)
	r.Delete("/{id}", DeleteHandler)
}
