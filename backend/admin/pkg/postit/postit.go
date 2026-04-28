package postit

import (
	"back-rex-admin/pkg/postit/gen"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"net/http"

	"github.com/go-chi/render"
)

type postPostitInput struct {
	MessageID string `json:"message_id"`
	Texte     string `json:"texte"`
}

func PostPostit(w http.ResponseWriter, r *http.Request) {
	var input postPostitInput
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if input.MessageID == "" || input.Texte == "" {
		services.InvalidRequestError(w, r, "message_id et texte requis", services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	userID := auth.GetSecurityUserId(r.Context())

	postit, err := gen.New(pgCtx.Db).InsertPostit(context.Background(), gen.InsertPostitParams{
		MessageID: input.MessageID,
		Texte:     input.Texte,
		AuteurID:  int32(*userID),
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, postit)
}

func GetPostits(w http.ResponseWriter, r *http.Request) {
	pgCtx := services.GetPgCtx(r.Context())
	postits, err := gen.New(pgCtx.Db).ListPostitsWithDetails(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if postits == nil {
		postits = []gen.PostitWithDetails{}
	}
	render.JSON(w, r, postits)
}
