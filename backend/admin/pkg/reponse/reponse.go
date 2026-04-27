package reponse

import (
	"back-rex-admin/pkg/reponse/gen"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type postReponseInput struct {
	FeedbackID int32  `json:"feedback_id"`
	Contenu    string `json:"contenu"`
}

func PostReponse(w http.ResponseWriter, r *http.Request) {
	var input postReponseInput
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if input.FeedbackID == 0 || input.Contenu == "" {
		services.InvalidRequestError(w, r, "feedback_id et contenu requis", services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	userID := auth.GetSecurityUserId(r.Context())

	user, err := auth.New(pgCtx.Db).GetUserById(context.Background(), int32(*userID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	reponse, err := gen.New(pgCtx.Db).InsertReponseByFeedbackId(context.Background(), gen.InsertReponseByFeedbackIdParams{
		ID:      input.FeedbackID,
		Contenu: input.Contenu,
		Auteur:  user.Name + " " + user.Surname,
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, reponse)
}

func GetReponses(w http.ResponseWriter, r *http.Request) {
	feedbackID, err := strconv.Atoi(chi.URLParam(r, "feedbackId"))
	if err != nil {
		services.InvalidRequestError(w, r, "feedbackId invalide", services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	reponses, err := gen.New(pgCtx.Db).ListReponsesByFeedbackId(context.Background(), int32(feedbackID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if reponses == nil {
		reponses = []gen.Reponse{}
	}
	render.JSON(w, r, reponses)
}
