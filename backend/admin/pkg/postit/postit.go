package postit

import (
	"back-rex-admin/pkg/postit/gen"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

type postPostitInput struct {
	MessageID     string `json:"message_id"`
	Reponse       string `json:"reponse"`
	MessageModere string `json:"message_modere"`
}

func PostPostit(w http.ResponseWriter, r *http.Request) {
	var input postPostitInput
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if input.MessageID == "" || input.Reponse == "" {
		services.InvalidRequestError(w, r, "message_id et reponse requis", services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	userID := auth.GetSecurityUserId(r.Context())

	postit, err := gen.New(pgCtx.Db).InsertPostit(context.Background(), gen.InsertPostitParams{
		MessageID:     input.MessageID,
		Reponse:       input.Reponse,
		MessageModere: pgtype.Text{String: input.MessageModere, Valid: input.MessageModere != ""},
		AuteurID:      int32(*userID),
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, postit)
}

func GetPostits(w http.ResponseWriter, r *http.Request) {
	months := int32(1)
	if m, err := strconv.Atoi(r.URL.Query().Get("months")); err == nil && m > 0 {
		months = int32(m)
	}
	promotion := r.URL.Query().Get("promotion")

	pgCtx := services.GetPgCtx(r.Context())
	postits, err := gen.New(pgCtx.Db).ListPostitsWithDetails(context.Background(), gen.ListPostitsWithDetailsParams{
		Months:    months,
		Promotion: promotion,
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if postits == nil {
		postits = []gen.ListPostitsWithDetailsRow{}
	}
	render.JSON(w, r, postits)
}

func DeletePostit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		services.InvalidRequestError(w, r, "id invalide", services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	if err := gen.New(pgCtx.Db).DeletePostit(context.Background(), int32(id)); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
