package pointage

import (
	"back-rex-common/pkg/auth"

	"github.com/go-chi/chi/v5"
)

// MakeRoutePointage monte les routes /pointage. Le montage (main.go) applique
// l'authentification seule ; le contrôle de rôle se fait ici, par sous-groupe :
// le pointage étudiant reste réservé aux élèves, les écrans prof/gestionnaire
// mobiles ont leurs propres routes.
func MakeRoutePointage() func(r chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRoles([]string{auth.RoleEleve}))
			r.Post("/", PostPointage())
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRoles([]string{auth.RoleProf, auth.RoleGestionnaire}))
			r.Get("/seances/jour", GetSeancesJourHandler)
			r.Get("/seance/{seanceId}/presence", GetSeancePresenceHandler)
			r.Post("/seance/{seanceId}/open", OpenSeanceProfHandler)
			r.Post("/seance/{seanceId}/close", CloseSeanceProfHandler)
			r.Get("/seance/{seanceId}/token", GetSeanceTokenHandler)
		})
	}
}
