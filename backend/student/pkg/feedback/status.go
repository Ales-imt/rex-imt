package feedback

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"unicode/utf8"

	"github.com/go-chi/render"
)

type statusRequest struct {
	MessageIDs []string `json:"message_ids"`
}

type statusResponse struct {
	MessageID       string  `json:"message_id"`
	ModerationStatus string `json:"moderation_status"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// feedbackStatusHandler renvoie le statut de modération (PENDING / PUBLISHED /
// REJECTED + motif éventuel) des messages dont l'étudiant fournit les
// message_id. La preuve de possession vient du couple (X-Pseudo, message_id) :
// aucun lien auteur→feedback n'est reconstruit côté serveur.
func feedbackStatusHandler(w http.ResponseWriter, r *http.Request) {
	pseudo := services.GetPseudo(r)
	if pseudo == "" {
		services.InvalidRequestError(w, r, "pseudo requis", services.NO_INFORMATION, nil)
		return
	}
	if utf8.RuneCountInString(pseudo) > maxPseudoLen {
		services.InvalidRequestError(w, r, "pseudo trop long", services.NO_INFORMATION, nil)
		return
	}

	var input statusRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if len(input.MessageIDs) == 0 {
		render.JSON(w, r, []statusResponse{})
		return
	}

	pool := services.GetPgCtx(r.Context()).Db
	rows, err := New(pool).GetFeedbackStatuses(context.Background(), GetFeedbackStatusesParams{
		Pseudo:  services.ToPgText(pseudo),
		Column2: input.MessageIDs,
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	out := make([]statusResponse, 0, len(rows))
	for _, row := range rows {
		item := statusResponse{
			MessageID:        row.MessageID.String,
			ModerationStatus: row.ModerationStatus,
		}
		if row.RejectionReason.Valid {
			reason := row.RejectionReason.String
			item.RejectionReason = &reason
		}
		out = append(out, item)
	}

	render.JSON(w, r, out)
}
