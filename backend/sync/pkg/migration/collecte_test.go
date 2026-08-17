package migration

import (
	"back-rex-sync/pkg/source"
	"errors"
	"testing"
	"time"
)

// sourceFragile enveloppe une sourceFixe et fait échouer, au choix, le planning
// de certaines promos ou la liste de certains groupes. C'est le mode de panne
// réel de webdfd : le référentiel répond, un flux ponctuel non.
type sourceFragile struct {
	sourceFixe
	planningKO map[string]bool
	groupeKO   map[string]bool
}

func (s sourceFragile) Planning(p0cle, debut, fin string) ([]source.Creneau, error) {
	if s.planningKO[p0cle] {
		return nil, errors.New("cgiempt indisponible")
	}
	return s.sourceFixe.Planning(p0cle, debut, fin)
}

func (s sourceFragile) MembresGroupe(grcle string) ([]string, string, error) {
	if s.groupeKO[grcle] {
		return nil, "", errors.New("flux listegroupe illisible")
	}
	return s.sourceFixe.MembresGroupe(grcle)
}

func acTest() anneeCourante {
	return anneeCourante{
		ID:    1,
		Annee: 2025,
		Debut: date(2025, time.September, 1),
		Fin:   date(2026, time.July, 31),
	}
}

func srcTest() sourceFixe {
	return sourceFixe{
		promos: []source.Promo{{ExternalID: "P1", Nom: "1A"}, {ExternalID: "P2", Nom: "2A"}},
		profs:  []source.Personne{{ExternalID: "PR1", Email: "a@x.fr"}},
		eleves: []source.Personne{{ExternalID: "EV1", Email: "e@x.fr"}},
		cours:  map[string]string{"10": "9.2.1 PROJET"},
		planning: map[string][]source.Creneau{
			"P1": {{Plcle: "pl1", P0cle: "P1", Cocle: "10", Grcle: "G1", Prcle: "PR1", Date: "20251001", HD: "0800", HF: "1000"}},
			"P2": {{Plcle: "pl2", P0cle: "P2", Cocle: "20", Grcle: "G2", Prcle: "PR2", Date: "20251002", HD: "0800", HF: "1000"}},
		},
		membres: map[string][]string{"G1": {"EV1"}, "G2": {"EV2"}},
	}
}

func TestCollecterNominal(t *testing.T) {
	c, err := collecter(srcTest(), acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}
	if len(c.promosOK) != 2 {
		t.Errorf("promosOK = %v, want les 2 promos", c.promosOK)
	}
	if len(c.creneaux) != 2 {
		t.Errorf("creneaux = %d, want 2", len(c.creneaux))
	}
	if len(c.groupesOK) != 2 {
		t.Errorf("groupesOK = %v, want les 2 groupes", c.groupesOK)
	}
	if got := c.cocles["10"].promoExtID; got != "P1" {
		t.Errorf("cocle 10 rattaché à %q, want P1", got)
	}
}

func TestCollecterPromoEnEchecSortDePromosOK(t *testing.T) {
	src := sourceFragile{sourceFixe: srcTest(), planningKO: map[string]bool{"P2": true}}

	c, err := collecter(src, acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: une promo inaccessible ne doit pas faire échouer le cycle: %v", err)
	}
	if !c.promosOK["P1"] {
		t.Error("P1 a répondu : elle doit être dans promosOK")
	}
	if c.promosOK["P2"] {
		t.Error("P2 est en échec : elle ne doit PAS être dans promosOK, sinon son planning serait annulé sur la foi d'un flux vide")
	}
	if _, vu := c.creneaux["pl2"]; vu {
		t.Error("le créneau de P2 ne doit pas avoir été collecté")
	}
	// Le prof et le groupe de P2 sortent mécaniquement du filtrage.
	if c.filtreProfs().retient("PR2") {
		t.Error("PR2 n'est connu que par le planning de P2, en échec : rien ne prouve qu'il enseigne")
	}
}

func TestCollecterGroupeEnEchecSortDeGroupesOK(t *testing.T) {
	src := sourceFragile{sourceFixe: srcTest(), groupeKO: map[string]bool{"G2": true}}

	c, err := collecter(src, acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}
	if !c.groupesOK["G1"] {
		t.Error("G1 a répondu : il doit être dans groupesOK")
	}
	if c.groupesOK["G2"] {
		t.Error("G2 est en échec : il ne doit PAS être dans groupesOK, sinon son effectif serait réconcilié à vide")
	}
	if c.filtreEleves().retient("EV2") {
		t.Error("EV2 n'est connu que par G2, en échec : aucun compte ne doit lui être ouvert sur cette base")
	}
}

func TestCollecterSansAnneeNInterrogePasLePlanning(t *testing.T) {
	c, err := collecter(srcTest(), anneeCourante{}, false, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}
	if len(c.creneaux) != 0 || len(c.promosOK) != 0 {
		t.Error("sans année courante, il n'y a pas de plage à interroger : le planning doit rester vide")
	}
	if len(c.profs) != 1 || len(c.eleves) != 1 {
		t.Error("les référentiels non datés doivent tout de même être collectés")
	}
}

func TestCollecterDetecteLeCoursMutualise(t *testing.T) {
	src := srcTest()
	// Le même COCLE tenu sous deux promos : sa période ne sera rattachée qu'à
	// la première, privant l'autre d'effectif attendu.
	src.planning["P2"] = []source.Creneau{
		{Plcle: "pl3", P0cle: "P2", Cocle: "10", Grcle: "G2", Date: "20251002", HD: "0800"},
	}

	c, err := collecter(src, acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}
	promos, signale := c.coclesMultiPromo["10"]
	if !signale {
		t.Fatal("un COCLE vu sous deux promos doit être signalé")
	}
	if len(promos) != 2 {
		t.Errorf("coclesMultiPromo[10] = %v, want les 2 promos", promos)
	}
}
