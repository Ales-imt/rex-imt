package feedback

import (
	"back-rex-admin/pkg/ia"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// FeedbackNotification est le payload reçu depuis le canal Postgres 'new_feedback'.
type FeedbackNotification struct {
	ID        int32   `json:"id"`
	Content   string  `json:"content"`
	UserID    int     `json:"user_id"`
	SeasonID  int32   `json:"season_id"`
	Promotion *string `json:"promotion"`
	Groupe    *string `json:"groupe"`
}

// ListenForNewFeedbacks écoute le canal Postgres 'new_feedback' et appelle
// CallbackFeedBack pour chaque feedback inséré par le serveur étudiant.
// Se reconnecte automatiquement en cas de perte de connexion.
// À lancer dans une goroutine au démarrage du serveur admin.
func ListenForNewFeedbacks(cfg *services.DatabaseConfig, connector ia.IAConnector) {
	dsn := services.ToDBS(cfg)
	for {
		if err := listenLoop(dsn, connector); err != nil {
			log.Printf("[feedback listener] connexion perdue: %v — reconnexion dans 5s", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func listenLoop(dsn string, connector ia.IAConnector) error {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "LISTEN new_feedback"); err != nil {
		return err
	}
	log.Println("[feedback listener] en écoute sur le canal 'new_feedback'")

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}

		var notif FeedbackNotification
		if err := json.Unmarshal([]byte(notification.Payload), &notif); err != nil {
			log.Printf("[feedback listener] payload invalide: %v", err)
			continue
		}

		CallbackFeedBack(notif, connector, conn)
	}
}
