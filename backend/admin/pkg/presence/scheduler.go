package presence

import (
	"back-rex-common/pkg/services"
	"context"
	"log"
	"time"
)

// anchorInterval : période de l'ancrage automatique. Elle borne aussi la
// fenêtre non couverte par le témoin externe : entre deux ancrages, seul le
// chaînage interne garantit l'intégrité (voir docs/temoin-externe.md).
const anchorInterval = 1 * time.Hour

// StartAnchorScheduler lance l'ancrage TSA automatique toutes les heures en
// arrière-plan, suivi de l'envoi du témoin externe pour chaque nouvelle ancre.
// Une première exécution est déclenchée immédiatement au démarrage.
//
// Si la tête de chaîne n'a pas bougé depuis le dernier passage, AnchorLast ne
// crée aucune ancre (idempotence par ledger_seq + tsa_url) et aucun e-mail
// n'est envoyé.
func StartAnchorScheduler(pgCfg *services.DatabaseConfig, cfg services.PresenceConfig) {
	if !cfg.Timestamp.Enabled {
		log.Println("[anchor] horodatage TSA désactivé, ancrage automatique inactif")
		return
	}

	pool := services.NewPG(context.Background(), services.ToDBS(pgCfg))

	go func() {
		runAnchor(pool, cfg)
		ticker := time.NewTicker(anchorInterval)
		defer ticker.Stop()
		for range ticker.C {
			runAnchor(pool, cfg)
		}
	}()
}

func runAnchor(pool *services.Postgres, cfg services.PresenceConfig) {
	ctx := context.Background()

	results, err := AnchorLast(ctx, pool.Db, cfg.Timestamp)
	if err != nil {
		log.Printf("[anchor] erreur ancrage automatique: %v", err)
		return
	}

	created := 0
	for _, res := range results {
		if res.Err != nil {
			log.Printf("[anchor] échec TSA %s: %v", res.TSAURL, res.Err)
			continue
		}
		if res.Created {
			created++
			log.Printf("[anchor] nouvelle ancre id=%d via %s", res.AnchorID, res.TSAURL)
		}
	}
	if created == 0 {
		return // tête de chaîne inchangée ou déjà ancrée : rien à témoigner
	}

	SendWitnesses(ctx, pool.Db, cfg.Witness, results)
}
