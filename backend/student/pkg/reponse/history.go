package reponse

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

type ChatMessage struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Source string `json:"source"`
	Ts     string `json:"ts"`
}

func GetChatHistory(w http.ResponseWriter, r *http.Request) {
	pseudo := services.GetPseudo(r)
	if pseudo == "" {
		services.InvalidRequestError(w, r, "pseudo requis", services.NO_INFORMATION, nil)
		return
	}

	months := 1
	if m := r.URL.Query().Get("months"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v > 0 {
			months = v
		}
	}

	since := time.Now().AddDate(0, -months, 0)

	const query = `
		SELECT f.message_id, f.content, 'me'::text, f.created_at
		FROM feedback f
		WHERE f.pseudo = $1 AND f.created_at >= $2
		UNION ALL
		SELECT 'reponse-' || r.id::text, r.contenu, 'other'::text, r.cree_le
		FROM reponse r
		JOIN feedback f ON r.message_id = f.message_id
		WHERE f.pseudo = $1 AND r.cree_le >= $2
		ORDER BY 4 ASC
	`
	pgCtx := services.GetPgCtx(r.Context())
	rows, err := pgCtx.Db.Query(context.Background(), query, pgtype.Text{String: pseudo, Valid: true}, since)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	defer rows.Close()

	messages := []ChatMessage{}
	for rows.Next() {
		var msg ChatMessage
		var ts time.Time
		if err := rows.Scan(&msg.ID, &msg.Text, &msg.Source, &ts); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		msg.Ts = ts.Format(time.RFC3339)
		messages = append(messages, msg)
	}

	render.JSON(w, r, messages)
}
