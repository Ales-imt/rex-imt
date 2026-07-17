package presence

import (
	"back-rex-common/pkg/presencetoken"
	"back-rex-common/pkg/services"

	"github.com/go-chi/chi/v5"
)

func RoutePresence(r chi.Router, cfg services.PresenceConfig) {
	presencetoken.SetSecret(cfg.TokenSecret)

	tsCfg := timestampCfg{
		tsCfg:      cfg.Timestamp,
		caCertPath: cfg.Timestamp.CaCertPath,
		witness:    cfg.Witness,
	}

	r.Get("/matieres/{matiereId}/planning", GetPlanningHandler)
	r.Post("/seance", OpenSeanceHandler)
	r.Post("/seance/{seanceId}/close", CloseSeanceHandler)
	r.Get("/seance/{seanceId}/token", GetTokenHandler)
	r.Get("/seance/{seanceId}/presence", GetPresenceHandler)
	r.Get("/matieres/{matiereId}/seances", ListSeancesHandler)
	r.Get("/matieres/{matiereId}/seance/slot", GetSlotHandler)
	r.Get("/seance/{seanceId}/pdf", PresencePdfHandler)

	r.Post("/ledger/anchor", LedgerAnchorHandler(tsCfg))
	r.Get("/ledger/verify", LedgerVerifyHandler(tsCfg))
	r.Get("/ledger/anchors", LedgerAnchorsHandler)
	r.Post("/ledger/verify-witness", WitnessVerifyHandler(tsCfg))
	r.Post("/witness/resend/{anchorID}", WitnessResendHandler(tsCfg))
}
