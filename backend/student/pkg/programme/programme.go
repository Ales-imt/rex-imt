package programme

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
)

// parisLoc est le fuseau des plannings (cf. migration/planning.go). La base
// tzdata est embarquée dans le binaire par student/cmd/main.go ; l'UTC ne sert
// que de garde-fou si la zone était malgré tout introuvable.
var parisLoc = func() *time.Location {
	l, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.UTC
	}
	return l
}()

const jourFmt = "20060102"

// parseJour lit une date AAAAMMJJ (les tirets éventuels sont tolérés, le front
// envoie AAAA-MM-JJ) et retourne minuit à Paris. ok vaut false si vide ou illisible.
func parseJour(s string) (time.Time, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), "-", "")
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(jourFmt, s, parisLoc)
	return t, err == nil
}

func getProgramme(connector ProgrammeConnector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetSecurityUserId(r.Context())
		pgCtx := services.GetPgCtx(r.Context())

		user, err := auth.New(pgCtx.Db).GetUserById(context.Background(), int32(*userID))
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		now := time.Now().In(parisLoc)
		debut, ok := parseJour(r.URL.Query().Get("start"))
		if !ok {
			debut = now.AddDate(0, 0, -7)
		}
		fin, ok := parseJour(r.URL.Query().Get("end"))
		if !ok {
			fin = now.AddDate(0, 1, 0)
		}
		// Le front envoie le dernier jour VOULU (le dimanche de la semaine
		// affichée) ; la borne haute des connecteurs est exclusive. Sans ces
		// 24 h, tout le dimanche disparaîtrait du planning.
		fin = fin.AddDate(0, 0, 1)

		cours, err := connector.GetProgramme(r.Context(), Demandeur{
			UserID: int32(*userID),
			Email:  user.Email,
			Roles:  user.Roles,
		}, debut, fin)
		if err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		render.JSON(w, r, cours)
	}
}
