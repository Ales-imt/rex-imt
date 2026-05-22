package user

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/user/gen"
	"context"
	"net/http"

	"github.com/go-chi/render"
)

func getMeHandler(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetSecurityUserId(r.Context())
	pgCtx := services.GetPgCtx(r.Context())

	me, err := gen.New(pgCtx.Db).GetMe(context.Background(), int32(*userID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, me)
}

func setInformedAtHandler(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetSecurityUserId(r.Context())
	pgCtx := services.GetPgCtx(r.Context())

	if err := gen.New(pgCtx.Db).SetInformedAt(context.Background(), int32(*userID)); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
