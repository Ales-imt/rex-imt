package rgpd

import (
	"back-rex-common/pkg/services"
	"context"
	"log"
	"time"
)

// anonymizeAfter : délai au-delà duquel les champs PII sont mis à NULL.
// deleteAfter    : délai au-delà duquel les enregistrements sont supprimés.
const (
	anonymizeAfter = 1 // an
	deleteAfter    = 3 // ans
)

// StartPurge lance la purge RGPD toutes les 24 h en arrière-plan.
// Une première exécution est déclenchée immédiatement au démarrage.
func StartPurge(cfg *services.DatabaseConfig) {
	pool := services.NewPG(context.Background(), services.ToDBS(cfg))
	go func() {
		runPurge(pool)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runPurge(pool)
		}
	}()
}

func runPurge(pool *services.Postgres) {
	ctx := context.Background()
	now := time.Now()
	log.Println("[rgpd] début de la purge")

	anonymizeThreshold := now.AddDate(-anonymizeAfter, 0, 0)
	deleteThreshold := now.AddDate(-deleteAfter, 0, 0)

	anonymizeFeedback(ctx, pool, anonymizeThreshold)
	anonymizeClassification(ctx, pool, anonymizeThreshold)
	anonymizeEvalVerbatim(ctx, pool, anonymizeThreshold)
	deleteClassification(ctx, pool, deleteThreshold)
	deleteFeedback(ctx, pool, deleteThreshold)

	log.Println("[rgpd] purge terminée")
}

func anonymizeFeedback(ctx context.Context, pool *services.Postgres, threshold time.Time) {
	tag, err := pool.Db.Exec(ctx, `
		UPDATE feedback SET
			strongbox  = NULL,
			pseudo     = NULL,
			message_id = NULL,
			promotion  = NULL,
			groupe     = NULL
		WHERE created_at < $1
		  AND pseudo IS NOT NULL
	`, threshold)
	if err != nil {
		log.Printf("[rgpd] erreur anonymisation feedback: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[rgpd] %d feedback(s) anonymisé(s)", tag.RowsAffected())
	}
}

func anonymizeClassification(ctx context.Context, pool *services.Postgres, threshold time.Time) {
	tag, err := pool.Db.Exec(ctx, `
		UPDATE feedback_classification fc SET
			promotion = NULL,
			groupe    = NULL
		FROM feedback f
		WHERE fc.feedback_id = f.id
		  AND f.created_at < $1
		  AND (fc.promotion IS NOT NULL OR fc.groupe IS NOT NULL)
	`, threshold)
	if err != nil {
		log.Printf("[rgpd] erreur anonymisation feedback_classification: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[rgpd] %d classification(s) anonymisée(s)", tag.RowsAffected())
	}
}

func deleteClassification(ctx context.Context, pool *services.Postgres, threshold time.Time) {
	tag, err := pool.Db.Exec(ctx, `
		DELETE FROM feedback_classification
		WHERE feedback_id IN (
			SELECT id FROM feedback WHERE created_at < $1
		)
	`, threshold)
	if err != nil {
		log.Printf("[rgpd] erreur suppression feedback_classification: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[rgpd] %d classification(s) supprimée(s)", tag.RowsAffected())
	}
}

func anonymizeEvalVerbatim(ctx context.Context, pool *services.Postgres, threshold time.Time) {
	tag, err := pool.Db.Exec(ctx, `
		UPDATE eval_verbatim SET strongbox = NULL
		WHERE created_at < $1
		  AND strongbox IS NOT NULL
	`, threshold)
	if err != nil {
		log.Printf("[rgpd] erreur anonymisation eval_verbatim: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[rgpd] %d verbatim(s) anonymisé(s)", tag.RowsAffected())
	}
}

func deleteFeedback(ctx context.Context, pool *services.Postgres, threshold time.Time) {
	tag, err := pool.Db.Exec(ctx, `
		DELETE FROM feedback WHERE created_at < $1
	`, threshold)
	if err != nil {
		log.Printf("[rgpd] erreur suppression feedback: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[rgpd] %d feedback(s) supprimé(s) (cascade reponse + postit)", tag.RowsAffected())
	}
}
