package feedback

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"

	"github.com/go-chi/render"
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
	// Réponse formatée avec items et itemCount
	render.JSON(w, r, feedbacks)
}
