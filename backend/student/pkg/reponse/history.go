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

// rejectedPlaceholder remplace le texte des feedbacks refusés dans l'historique :
// au refus, raw_content est écrasé par sa version chiffrée (age), qui ne doit pas
// sortir du serveur.
const rejectedPlaceholder = "[message refusé par la modération]"

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

	pgCtx := services.GetPgCtx(r.Context())
	rows, err := New(pgCtx.Db).GetChatHistoryByPseudo(context.Background(), GetChatHistoryByPseudoParams{
		RejectedPlaceholder: rejectedPlaceholder,
		Pseudo:              pgtype.Text{String: pseudo, Valid: true},
		Since:               pgtype.Timestamptz{Time: since, Valid: true},
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	messages := []ChatMessage{}
	for _, row := range rows {
		messages = append(messages, ChatMessage{
			ID:     row.ID.String,
			Text:   row.Text,
			Source: row.Source,
			Ts:     row.Ts.Time.Format(time.RFC3339),
		})
	}

	render.JSON(w, r, messages)
}
