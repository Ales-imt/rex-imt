package planning

import "github.com/go-chi/chi/v5"

func RoutePlanning(r chi.Router) {
	r.Get("/reservation", GetReservations)
	r.Get("/heures", GetHeures)
	r.Get("/salles/occupation", GetOccupation)
	r.Get("/salles/creneaux", GetCreneaux)
	r.Get("/salles/non-resolues", GetNonResolues)
}
