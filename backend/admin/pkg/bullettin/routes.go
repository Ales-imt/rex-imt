package bullettin

import (
	"time"

	"back-rex-common/pkg/services"

	"github.com/go-chi/chi/v5"
)

// handlers porte la configuration et l'état partagé des routes : URL du service
// Gotenberg (conversion docx -> PDF) et le suivi des jobs de génération.
type handlers struct {
	gotenbergURL string
	jobs         *jobStore
}

// RouteBullettin monte les routes de génération des bulletins sur le routeur
// fourni (préfixe /bullettin défini par l'appelant dans cmd/main.go).
func RouteBullettin(r chi.Router, cfg services.BullettinConfig) {
	h := &handlers{
		gotenbergURL: cfg.GotenbergURL,
		jobs:         newJobStore(30 * time.Minute),
	}
	r.Post("/sheets", h.handleSheets)
	r.Post("/columns", h.handleColumns)
	r.Post("/generate", h.handleGenerateStart)
	r.Get("/generate/{id}/status", h.handleGenerateStatus)
	r.Get("/generate/{id}/download", h.handleGenerateDownload)
}
