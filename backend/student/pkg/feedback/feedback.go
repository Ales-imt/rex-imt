package feedback

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/service"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
)

type FeedbackRequest struct {
	Content   string `json:"content"`
	Pseudo    string `json:"pseudo"`
	MessageID string `json:"message_id"`
	Promotion string `json:"promotion"`
	Groupe    string `json:"groupe"`
}

func makeFeedbackHandler(agePublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var inputs []FeedbackRequest
		if err := render.DecodeJSON(r.Body, &inputs); err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		if err := InsertFeedbacks(r, inputs, agePublicKey); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func InsertFeedbacks(r *http.Request, inputs []FeedbackRequest, agePublicKey string) error {
	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db)
	now := time.Now()

	userID := auth.GetSecurityUserId(r.Context())
	studentID := *userID

	strongbox, err := service.EncryptStrongbox(agePublicKey, r, studentID)
	if err != nil {
		return fmt.Errorf("échec chiffrement strongbox : %w", err)
	}

	for _, input := range inputs {
		_, err := queries.InsertFeedbacks(context.Background(), InsertFeedbacksParams{
			Content:   input.Content,
			CreatedAt: services.ToPgTimestamptz(&now),
			Strongbox: services.ToPgText(strongbox),
			Pseudo:    services.ToPgText(input.Pseudo),
			MessageID: services.ToPgText(input.MessageID),
			Promotion: services.ToPgText(input.Promotion),
			Groupe:    services.ToPgText(input.Groupe),
		})
		if err != nil {
			return fmt.Errorf("échec insertion feedback : %w", err)
		}
	}
	return nil
}
