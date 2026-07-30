package programme

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/render"
)

func getProgramme(connector ProgrammeConnector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetSecurityUserId(r.Context())
		pgCtx := services.GetPgCtx(r.Context())

		user, err := auth.New(pgCtx.Db).GetUserById(context.Background(), int32(*userID))
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		start := strings.ReplaceAll(r.URL.Query().Get("start"), "-", "")
		end := strings.ReplaceAll(r.URL.Query().Get("end"), "-", "")
		if start == "" {
			start = time.Now().AddDate(0, 0, -7).Format("20060102")
		}
		if end == "" {
			end = time.Now().AddDate(0, 1, 0).Format("20060102")
		}

		gestionnaire := slices.Contains(user.Roles, auth.RoleGestionnaire)

		cours, err := connector.GetProgramme(r.Context(), user.Email, start, end, gestionnaire)
		if err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		render.JSON(w, r, cours)
	}
}
