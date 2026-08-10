package justification

// Bornes temporelles d'une excuse : parsing, validation, conversion en
// tstzrange. Fonctions pures, testées unitairement — c'est ici que se joue la
// correction des plages à cheval sur un changement d'heure.

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// dureeMax : au-delà, la saisie est refusée. Une excuse d'un an et demi n'est
// pas une longue maladie, c'est une faute de frappe sur l'année. Le garde-fou
// est doublé côté UI par une confirmation explicite au-delà de 30 jours.
const dureeMax = 400 * 24 * time.Hour

// parisLoc : les champs datetime du dialogue produisent des heures locales et
// les plannings sont exprimés en heure de Paris ; tstzrange stocke de l'UTC.
// La conversion doit donc être explicite — un time.Parse sans localisation
// interpréterait « 18:00 » en UTC et déplacerait la borne d'une ou deux heures
// selon la saison. tzdata est embarqué dans le binaire (cf. cmd/main.go).
var parisLoc = func() *time.Location {
	l, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.UTC
	}
	return l
}()

// formesLocales : formats acceptés pour une heure locale sans fuseau, tels que
// les émet un <input type="datetime-local">.
var formesLocales = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
}

// parseParis lit une borne de plage. Un horodatage portant son fuseau (RFC 3339)
// est pris tel quel ; une heure locale nue est interprétée en heure de Paris.
func parseParis(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("horodatage manquant")
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, f := range formesLocales {
		if t, err := time.ParseInLocation(f, s, parisLoc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("horodatage invalide : %q", s)
}

// makePeriode valide le couple (début, fin) et le convertit en tstzrange
// semi-ouvert [début, fin) — même convention que le && de ListSeancesCouvertes :
// une séance qui commence exactement à la fin de l'excuse n'est pas couverte.
func makePeriode(debutStr, finStr string) (pgtype.Range[pgtype.Timestamptz], error) {
	var vide pgtype.Range[pgtype.Timestamptz]

	debut, err := parseParis(debutStr)
	if err != nil {
		return vide, fmt.Errorf("début : %w", err)
	}
	fin, err := parseParis(finStr)
	if err != nil {
		return vide, fmt.Errorf("fin : %w", err)
	}
	if !fin.After(debut) {
		return vide, fmt.Errorf("la fin doit être postérieure au début")
	}
	if fin.Sub(debut) > dureeMax {
		return vide, fmt.Errorf("plage de %d jours : au-delà de %d jours, la saisie est refusée (année erronée ?)",
			int(fin.Sub(debut).Hours()/24), int(dureeMax.Hours()/24))
	}

	return pgtype.Range[pgtype.Timestamptz]{
		Lower:     pgtype.Timestamptz{Time: debut.UTC(), Valid: true},
		Upper:     pgtype.Timestamptz{Time: fin.UTC(), Valid: true},
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
		Valid:     true,
	}, nil
}
