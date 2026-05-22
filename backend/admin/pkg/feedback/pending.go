package feedback

import (
	"back-rex-admin/pkg/ia"
	"back-rex-common/pkg/services"
	"context"
	"log"
)

// ProcessPendingFeedbacks récupère tous les feedbacks sans classification IA
// et les traite via CallbackFeedBack. À lancer en goroutine au démarrage.
func ProcessPendingFeedbacks(cfg *services.DatabaseConfig, connector ia.IAConnector) {
	ctx := context.Background()
	dsn := services.ToDBS(cfg)
	pool := services.NewPG(ctx, dsn)

	rows, err := New(pool.Db).ListPendingFeedbacks(ctx)
	if err != nil {
		log.Printf("[pending feedbacks] erreur requête: %v", err)
		return
	}

	if len(rows) == 0 {
		log.Println("[pending feedbacks] aucun feedback en attente")
		return
	}

	log.Printf("[pending feedbacks] %d feedback(s) à traiter", len(rows))
	for _, row := range rows {
		notif := rowToNotification(row)
		CallbackFeedBack(notif, connector, pool.Db)
	}
	log.Println("[pending feedbacks] traitement terminé")
}

func rowToNotification(row ListPendingFeedbacksRow) FeedbackNotification {
	notif := FeedbackNotification{
		ID:      row.ID,
		Content: row.Content,
	}
	if row.Promotion.Valid {
		notif.Promotion = &row.Promotion.String
	}
	if row.Groupe.Valid {
		notif.Groupe = &row.Groupe.String
	}
	return notif
}
