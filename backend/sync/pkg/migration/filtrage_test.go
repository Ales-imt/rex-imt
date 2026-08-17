package migration

import (
	"back-rex-sync/pkg/source"
	"testing"
)

// collecteAvec construit une collecte minimale : un planning d'une promo tenu
// par un prof, et un groupe peuplé de deux élèves.
func collecteAvec(promoOK, groupeOK bool) *collecte {
	c := &collecte{
		promos:    []source.Promo{{ExternalID: "P1"}},
		creneaux:  map[string]source.Creneau{"pl1": {Cocle: "10", Grcle: "G1", Prcle: "PR1"}},
		grcles:    map[string]grcleInfo{"G1": {promoExtID: "P1"}},
		promosOK:  map[string]bool{},
		membres:   map[string][]string{"G1": {"EV1", "EV2"}},
		groupesOK: map[string]bool{},
	}
	if promoOK {
		c.promosOK["P1"] = true
	}
	if groupeOK {
		c.groupesOK["G1"] = true
	}
	return c
}

func TestFiltreProfs(t *testing.T) {
	t.Run("actif : retient le prof du planning, écarte l'inconnu", func(t *testing.T) {
		f := collecteAvec(true, true).filtreProfs()
		if !f.actif {
			t.Fatal("le filtre doit être actif quand une promo a répondu")
		}
		if !f.retient("PR1") {
			t.Error("PR1 tient un créneau : il doit être retenu")
		}
		if f.retient("PR9") {
			t.Error("PR9 ne tient aucun créneau : aucun compte ne doit lui être ouvert")
		}
	})

	t.Run("inactif quand aucune promo n'a répondu", func(t *testing.T) {
		// Pas d'information sur qui enseigne : on ne filtre pas, exactement
		// comme seancesPerimees n'annule rien dans le même cas.
		f := collecteAvec(false, true).filtreProfs()
		if f.actif {
			t.Fatal("le filtre doit s'effacer sans planning exploitable")
		}
		if !f.retient("PR9") {
			t.Error("filtre inactif : tout le monde doit passer")
		}
	})
}

func TestFiltreEleves(t *testing.T) {
	t.Run("actif : retient les membres des groupes lus", func(t *testing.T) {
		f := collecteAvec(true, true).filtreEleves()
		if !f.actif {
			t.Fatal("le filtre doit être actif quand une liste de groupe a été lue")
		}
		for _, ev := range []string{"EV1", "EV2"} {
			if !f.retient(ev) {
				t.Errorf("%s est membre de G1 : il doit être retenu", ev)
			}
		}
		if f.retient("EV9") {
			t.Error("EV9 n'est dans aucun groupe du planning : aucun compte ne doit lui être ouvert")
		}
	})

	t.Run("inactif quand aucune liste n'a été récupérée", func(t *testing.T) {
		f := collecteAvec(true, false).filtreEleves()
		if f.actif {
			t.Fatal("le filtre doit s'effacer sans liste de groupe exploitable")
		}
		if !f.retient("EV9") {
			t.Error("filtre inactif : tout le monde doit passer")
		}
	})

	t.Run("les membres d'un groupe en échec ne comptent pas", func(t *testing.T) {
		// membres["G1"] est renseigné mais groupesOK ne l'est pas : la liste
		// vient d'un cycle où le flux a échoué, elle ne prouve rien.
		c := collecteAvec(true, false)
		c.groupesOK["G2"] = true
		c.membres["G2"] = []string{"EV3"}
		f := c.filtreEleves()
		if !f.retient("EV3") {
			t.Error("EV3 vient d'un groupe lu avec succès : il doit être retenu")
		}
		if f.retient("EV1") {
			t.Error("EV1 vient d'un groupe en échec : il ne doit pas être retenu")
		}
	})
}

func TestPrclesVusIgnoreLesCreneauxSansProf(t *testing.T) {
	c := &collecte{creneaux: map[string]source.Creneau{
		"pl1": {Prcle: "PR1"},
		"pl2": {Prcle: ""},
		"pl3": {Prcle: "PR1"},
	}}
	vus := c.prclesVus()
	if len(vus) != 1 || !vus["PR1"] {
		t.Errorf("prclesVus = %v, want {PR1}", vus)
	}
}
