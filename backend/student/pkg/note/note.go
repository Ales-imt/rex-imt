package note

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"net/http"

	"github.com/go-chi/render"
)

func getNotes(connector NoteConnector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetSecurityUserId(r.Context())
		pgCtx := services.GetPgCtx(r.Context())

		user, err := auth.New(pgCtx.Db).GetUserById(context.Background(), int32(*userID))
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		result, err := connector.GetNotes(r.Context(), user.Email)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		render.JSON(w, r, result)
	}
}
