package feedback

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func anonymizeFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	pseudo := r.Header.Get("X-Pseudo")
	if pseudo == "" {
		services.InvalidRequestError(w, r, "pseudo requis", services.NO_INFORMATION, nil)
		return
	}

	messageID := chi.URLParam(r, "messageID")
	if messageID == "" {
		services.InvalidRequestError(w, r, "messageID requis", services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	if err := New(pgCtx.Db).AnonymizeFeedback(context.Background(), AnonymizeFeedbackParams{
		MessageID: services.ToPgText(messageID),
		Pseudo:    services.ToPgText(pseudo),
	}); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
