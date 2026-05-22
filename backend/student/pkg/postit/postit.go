package postit

import (
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/postit/gen"
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

func GetPromotions(w http.ResponseWriter, r *http.Request) {
	pgCtx := services.GetPgCtx(r.Context())
	promotions, err := gen.New(pgCtx.Db).ListPostitsPromotions(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if promotions == nil {
		promotions = []pgtype.Text{}
	}
	render.JSON(w, r, promotions)
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
