package justification

// Tests des bornes de plage. Sans base : ce sont des fonctions pures, et c'est
// ici que se joue le fuseau horaire — la borne saisie « 18:00 » doit rester
// 18:00 heure de Paris quelle que soit la saison.

import (
	"github.com/jackc/pgx/v5/pgtype"
	"testing"
	"time"
)

func TestParseParisRespecteLeFuseau(t *testing.T) {
	cas := []struct {
		nom      string
		entree   string
		attendu  string // instant attendu, en UTC
		decalage string
	}{
		{"heure d'hiver (CET, UTC+1)", "2026-03-28T18:00", "2026-03-28T17:00:00Z", "+01"},
		{"heure d'été (CEST, UTC+2)", "2026-03-30T18:00", "2026-03-30T16:00:00Z", "+02"},
		{"veille du retour à l'heure d'hiver", "2026-10-24T18:00", "2026-10-24T16:00:00Z", "+02"},
		{"lendemain du retour à l'heure d'hiver", "2026-10-26T18:00", "2026-10-26T17:00:00Z", "+01"},
		{"secondes acceptées", "2026-03-30T18:00:00", "2026-03-30T16:00:00Z", "+02"},
		{"horodatage portant déjà son fuseau", "2026-03-30T18:00:00+02:00", "2026-03-30T16:00:00Z", "+02"},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := parseParis(c.entree)
			if err != nil {
				t.Fatalf("parseParis(%q): %v", c.entree, err)
			}
			if s := got.UTC().Format(time.RFC3339); s != c.attendu {
				t.Errorf("parseParis(%q) = %s, attendu %s (décalage %s)", c.entree, s, c.attendu, c.decalage)
			}
		})
	}
}

func TestParseParisRejetteUneSaisieIllisible(t *testing.T) {
	for _, entree := range []string{"", "hier", "2026-13-45T99:99", "30/03/2026 18:00"} {
		if _, err := parseParis(entree); err == nil {
			t.Errorf("parseParis(%q) accepté, attendu une erreur", entree)
		}
	}
}

// La plage traverse le changement d'heure du 29 mars 2026 : la borne de fin
// doit valoir 16:00 UTC (18:00 CEST), ni 17:00 (une heure de trop) ni 15:00.
func TestMakePeriodeAChevalSurLeChangementDHeure(t *testing.T) {
	p, err := makePeriode("2026-03-28T08:00", "2026-03-30T18:00")
	if err != nil {
		t.Fatalf("makePeriode: %v", err)
	}
	if got := p.Lower.Time.UTC().Format(time.RFC3339); got != "2026-03-28T07:00:00Z" {
		t.Errorf("borne basse = %s, attendu 2026-03-28T07:00:00Z (08:00 CET)", got)
	}
	if got := p.Upper.Time.UTC().Format(time.RFC3339); got != "2026-03-30T16:00:00Z" {
		t.Errorf("borne haute = %s, attendu 2026-03-30T16:00:00Z (18:00 CEST)", got)
	}
	// Semi-ouverte : une séance commençant exactement à la borne n'est pas couverte.
	if p.UpperType != pgtype.Exclusive {
		t.Errorf("borne haute de type %v, attendue exclusive", p.UpperType)
	}
}

func TestMakePeriodeRejetteLesPlagesAberrantes(t *testing.T) {
	cas := []struct {
		nom         string
		debut, fin  string
		doitEchouer bool
	}{
		{"plage normale", "2026-05-04T08:00", "2026-05-06T18:00", false},
		{"plage inversée", "2026-05-06T18:00", "2026-05-04T08:00", true},
		{"plage nulle", "2026-05-04T08:00", "2026-05-04T08:00", true},
		{"longue maladie (60 jours)", "2026-01-05T08:00", "2026-03-05T18:00", false},
		{"année tapée de travers (> 400 jours)", "2025-05-04T08:00", "2026-09-04T18:00", true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, err := makePeriode(c.debut, c.fin)
			if c.doitEchouer && err == nil {
				t.Errorf("makePeriode(%s, %s) accepté, attendu un refus", c.debut, c.fin)
			}
			if !c.doitEchouer && err != nil {
				t.Errorf("makePeriode(%s, %s) refusé : %v", c.debut, c.fin, err)
			}
		})
	}
}
