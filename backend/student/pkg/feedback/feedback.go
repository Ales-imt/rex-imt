package feedback

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/service"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxContentLen = 2000
	maxPseudoLen  = 100
)

type FeedbackRequest struct {
	Content   string `json:"content"`
	Pseudo    string `json:"pseudo"`
	MessageID string `json:"message_id"`
}

func (f FeedbackRequest) validate() error {
	if utf8.RuneCountInString(f.Content) > maxContentLen {
		return fmt.Errorf("content dépasse %d caractères", maxContentLen)
	}
	if utf8.RuneCountInString(f.Pseudo) > maxPseudoLen {
		return fmt.Errorf("pseudo dépasse %d caractères", maxPseudoLen)
	}
	return nil
}

func makeFeedbackHandler(agePublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var inputs []FeedbackRequest
		if err := render.DecodeJSON(r.Body, &inputs); err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		for _, input := range inputs {
			if err := input.validate(); err != nil {
				services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}
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

	nameSurname, err := queries.GetUserNameSurname(context.Background(), int32(studentID))
	if err != nil {
		return fmt.Errorf("échec récupération nom/prénom : %w", err)
	}

	strongbox, err := service.EncryptStrongbox(agePublicKey, r, studentID, nameSurname.Name, nameSurname.Surname)
	if err != nil {
		return fmt.Errorf("échec chiffrement strongbox : %w", err)
	}

	var promotion, groupe pgtype.Text
	promoGroupe, err := queries.GetStudentPromoAndGroupes(context.Background(), int32(studentID))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("échec récupération promotion/groupe : %w", err)
	}
	if err == nil {
		promotion = promoGroupe.Promotion
		groupe = services.ToPgText(promoGroupe.Groupes)
	}

	for _, input := range inputs {
		_, err := queries.InsertFeedbacks(context.Background(), InsertFeedbacksParams{
			Content:   input.Content,
			CreatedAt: services.ToPgTimestamptz(&now),
			Strongbox: services.ToPgText(strongbox),
			Pseudo:    services.ToPgText(input.Pseudo),
			MessageID: services.ToPgText(input.MessageID),
			Promotion: promotion,
			Groupe:    groupe,
		})
		if err != nil {
			return fmt.Errorf("échec insertion feedback : %w", err)
		}
	}
	return nil
}
