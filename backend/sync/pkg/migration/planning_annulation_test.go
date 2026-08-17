package migration

import (
	"slices"
	"testing"
	"time"
)

// Ces tests portent sur la SEULE décision risquée de l'annulation : elle
// s'appuie sur l'absence d'une donnée, et l'absence peut aussi bien vouloir dire
// « supprimé en amont » que « pas reçu ». Confondre les deux annulerait des
// séances bien réelles, avec leurs feuilles de présence.

func jour(j int) time.Time {
	return time.Date(2026, time.March, j, 8, 0, 0, 0, time.UTC)
}

var (
	debutPlage = jour(1)
	finPlage   = jour(31)
)

func TestSeancesPerimees(t *testing.T) {
	// Deux promos : 10 récupérée avec succès, 20 en échec réseau.
	candidates := []seanceCandidate{
		{id: 1, plcle: "PL1", promoID: 10, startsAt: jour(10), hasStart: true},
		{id: 2, plcle: "PL2", promoID: 20, startsAt: jour(10), hasStart: true},
		{id: 3, plcle: "PL3", promoID: 10, startsAt: jour(10), hasStart: true},
		{id: 4, plcle: "PL4", promoID: 10, startsAt: jour(40), hasStart: true},
		{id: 5, plcle: "PL5", promoID: 0, startsAt: jour(10), hasStart: true},
		{id: 6, plcle: "PL6", promoID: 10, hasStart: false},
	}

	cases := []struct {
		nom      string
		vus      map[string]struct{}
		promosOK map[int64]bool
		want     []int64
	}{
		{
			nom:      "séance disparue d'une promo récupérée : annulée",
			vus:      map[string]struct{}{},
			promosOK: map[int64]bool{10: true, 20: true},
			// PL2 : promo 20 marquée OK ici, donc annulée elle aussi.
			want: []int64{1, 2, 3},
		},
		{
			nom:      "promo en échec : ses séances restent intactes",
			vus:      map[string]struct{}{},
			promosOK: map[int64]bool{10: true},
			want:     []int64{1, 3},
		},
		{
			nom:      "aucune promo récupérée : on n'annule rien",
			vus:      map[string]struct{}{},
			promosOK: map[int64]bool{},
			want:     nil,
		},
		{
			nom:      "créneau revu pendant le cycle : intact",
			vus:      map[string]struct{}{"PL1": {}, "PL3": {}},
			promosOK: map[int64]bool{10: true},
			want:     nil,
		},
	}

	for _, tc := range cases {
		got := seancesPerimees(candidates, tc.vus, tc.promosOK, debutPlage, finPlage)
		slices.Sort(got)
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s : seancesPerimees = %v, attendu %v", tc.nom, got, tc.want)
		}
	}
}

// Une séance hors de la plage réellement interrogée n'a pas pu être confirmée
// par ce cycle : le flux ne disait rien à son sujet. PL4 (au-delà de la borne
// haute) et PL6 (sans horaire, donc impossible à situer) doivent survivre.
func TestSeancesPerimeesHorsPlage(t *testing.T) {
	candidates := []seanceCandidate{
		{id: 4, plcle: "PL4", promoID: 10, startsAt: jour(40), hasStart: true},
		{id: 6, plcle: "PL6", promoID: 10, hasStart: false},
		{id: 7, plcle: "PL7", promoID: 10, startsAt: jour(-5), hasStart: true},
	}
	got := seancesPerimees(candidates, map[string]struct{}{}, map[int64]bool{10: true}, debutPlage, finPlage)
	if len(got) != 0 {
		t.Errorf("seancesPerimees = %v, attendu aucune annulation hors plage", got)
	}
}

// public.annee.fin est une `date` : elle arrive à minuit. Comparée telle
// quelle, elle excluait toutes les séances du dernier jour de l'année scolaire,
// qui n'étaient donc jamais annulables — alors même que le planning avait bien
// été interrogé jusqu'à cette date incluse (Format("20060102") sur la borne).
func TestSeancesPerimeesDernierJourDeLAnnee(t *testing.T) {
	debut := date(2025, time.September, 1)
	fin := date(2026, time.July, 31) // minuit, comme la colonne `date`

	candidates := []seanceCandidate{
		{id: 1, plcle: "PL1", promoID: 10, startsAt: time.Date(2026, time.July, 31, 8, 0, 0, 0, time.UTC), hasStart: true},
		{id: 2, plcle: "PL2", promoID: 10, startsAt: time.Date(2026, time.July, 31, 23, 59, 0, 0, time.UTC), hasStart: true},
		{id: 3, plcle: "PL3", promoID: 10, startsAt: time.Date(2026, time.August, 1, 8, 0, 0, 0, time.UTC), hasStart: true},
	}

	got := seancesPerimees(candidates, map[string]struct{}{}, map[int64]bool{10: true}, debut, fin)
	slices.Sort(got)
	if !slices.Equal(got, []int64{1, 2}) {
		t.Errorf("seancesPerimees = %v, attendu [1 2] : le dernier jour est dans la plage interrogée, le lendemain non", got)
	}
}

// Une séance sans promotion (promotion_id NULL, donc promoID 0) ne peut être
// rattachée à aucun flux : rien ne prouve qu'elle a disparu.
func TestSeancesPerimeesSansPromotion(t *testing.T) {
	candidates := []seanceCandidate{
		{id: 5, plcle: "PL5", promoID: 0, startsAt: jour(10), hasStart: true},
	}
	got := seancesPerimees(candidates, map[string]struct{}{}, map[int64]bool{10: true, 0: true}, debutPlage, finPlage)
	if len(got) != 1 {
		// promosOK[0] explicitement vrai : le garde-fou ne doit pas dépendre
		// d'un traitement spécial de la valeur 0, mais bien de l'appartenance.
		t.Errorf("seancesPerimees = %v, attendu [5] quand la promo 0 est déclarée OK", got)
	}
	got = seancesPerimees(candidates, map[string]struct{}{}, map[int64]bool{10: true}, debutPlage, finPlage)
	if len(got) != 0 {
		t.Errorf("seancesPerimees = %v, attendu aucune annulation sans promotion rattachée", got)
	}
}
