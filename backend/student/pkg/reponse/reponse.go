package reponse

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

func GetUnreadReponses(w http.ResponseWriter, r *http.Request) {
	pseudo := r.Header.Get("X-Pseudo")
	if pseudo == "" {
		services.InvalidRequestError(w, r, "pseudo requis", services.NO_INFORMATION, nil)
		return
	}

	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db)

	reponses, err := queries.GetUnreadReponsesByPseudo(context.Background(), pgtype.Text{String: pseudo, Valid: true})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if err := queries.MarkReponsesByPseudoAsRead(context.Background(), pgtype.Text{String: pseudo, Valid: true}); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, reponses)
}
