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

	pool := services.GetPgCtx(r.Context()).Db
	tx, err := pool.Begin(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	defer tx.Rollback(context.Background())

	q := New(tx)

	if err := q.AnonymizeFeedback(context.Background(), AnonymizeFeedbackParams{
		MessageID: services.ToPgText(messageID),
		Pseudo:    services.ToPgText(pseudo),
	}); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if err := q.AnonymizePostitByFeedback(context.Background(), messageID); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
