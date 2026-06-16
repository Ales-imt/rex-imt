package feedback

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

type Classification struct {
	FeedbackID int    `json:"feedback_id"`
	Categorie  string `json:"categorie"`
	SousCateg  string `json:"sous_categorie"`
	Sentiment  string `json:"sentiment"` // positif | negatif | neutre | mitige
	Urgence    int    `json:"urgence"`   // 1 (faible) à 5 (critique)
	Resume     string `json:"resume"`
}

func GetAllFeedback(w http.ResponseWriter, r *http.Request) {
	pgctx := services.GetPgCtx(r.Context())
	query := New(pgctx.Db)

	feedbacks, err := query.ListFeedbacksWithClassification(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if feedbacks == nil {
		feedbacks = []ListFeedbacksWithClassificationRow{}
	}
	if !auth.HasRole(r.Context(), auth.RoleAdmin) {
		for i := range feedbacks {
			feedbacks[i].Strongbox = pgtype.Text{}
		}
	}
	render.JSON(w, r, feedbacks)
}

func GetRecentFeedback(w http.ResponseWriter, r *http.Request) {
	months := int32(1)
	if m, err := strconv.Atoi(r.URL.Query().Get("months")); err == nil && m > 0 {
		months = int32(m)
	}

	pgctx := services.GetPgCtx(r.Context())
	query := New(pgctx.Db)

	feedbacks, err := query.ListRecentFeedbacksWithClassification(context.Background(), months)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if feedbacks == nil {
		feedbacks = []ListRecentFeedbacksWithClassificationRow{}
	}
	if !auth.HasRole(r.Context(), auth.RoleAdmin) {
		for i := range feedbacks {
			feedbacks[i].Strongbox = pgtype.Text{}
		}
	}
	render.JSON(w, r, feedbacks)
}
