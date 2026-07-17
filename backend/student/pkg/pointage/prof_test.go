package pointage

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// canManageSeance est la décision d'accès par séance : le SQL filtre déjà les
// listes (ListSeancesJourByProf), cette fonction protège les routes unitaires
// (presence, open, close, token).
func TestCanManageSeance(t *testing.T) {
	profID := func(id int32) pgtype.Int4 { return pgtype.Int4{Int32: id, Valid: true} }
	noProf := pgtype.Int4{}

	cases := []struct {
		name           string
		isGestionnaire bool
		userID         int
		profID         pgtype.Int4
		want           bool
	}{
		{"gestionnaire accède à toute séance", true, 99, profID(7), true},
		{"gestionnaire accède même sans prof_id", true, 99, noProf, true},
		{"prof accède à sa séance", false, 7, profID(7), true},
		{"prof refusé sur la séance d'un autre", false, 8, profID(7), false},
		{"prof refusé si séance sans prof_id", false, 7, noProf, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canManageSeance(tc.isGestionnaire, tc.userID, tc.profID); got != tc.want {
				t.Errorf("canManageSeance(%v, %d, %+v) = %v, want %v",
					tc.isGestionnaire, tc.userID, tc.profID, got, tc.want)
			}
		})
	}
}
