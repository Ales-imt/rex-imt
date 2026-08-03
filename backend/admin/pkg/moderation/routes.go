package moderation

import "github.com/go-chi/chi/v5"

func RouteModeration(agePublicKey string) func(r chi.Router) {
	return func(r chi.Router) {
		// Feedbacks (clé entière).
		r.Get("/pending", ListPending)
		r.Post("/{id}/approve", Approve)
		r.Post("/{id}/reject", makeReject(agePublicKey))

		// Verbatims d'évaluation (clé uuid), même pipeline.
		r.Route("/verbatim", func(r chi.Router) {
			r.Get("/pending", ListPendingVerbatim)
			r.Post("/{id}/approve", ApproveVerbatim)
			r.Post("/{id}/reject", makeRejectVerbatim(agePublicKey))
		})
	}
}
