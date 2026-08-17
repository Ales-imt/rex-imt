package migration

import (
	"back-rex-sync/pkg/source"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// planningAvecPertes fabrique un flux qui exerce chaque point d'écartement de
// la collecte, plus un créneau parfaitement valide en témoin.
func planningAvecPertes() sourceFixe {
	s := srcTest()
	s.planning["P1"] = []source.Creneau{
		{Plcle: "ok1", P0cle: "P1", Cocle: "10", Grcle: "G1", Date: "20251001", HD: "0800", HF: "1000"},
		{Plcle: "x1", P0cle: "P1", Cocle: "", Date: "20251001", HD: "0800"},  // COCLE vide
		{Plcle: "x2", P0cle: "P1", Cocle: "0", Date: "20251001", HD: "0800"}, // COCLE fictif
		{Plcle: "", P0cle: "P1", Cocle: "10", Date: "20251001", HD: "1000"},  // PL vide
	}
	// Même PL rendu sous deux promotions : le second écrase le premier.
	s.planning["P2"] = []source.Creneau{
		{Plcle: "ok1", P0cle: "P2", Cocle: "20", Grcle: "G2", Date: "20251002", HD: "0800", HF: "1000"},
	}
	return s
}

func TestCollecteTraceLesCreneauxEcartes(t *testing.T) {
	c, err := collecter(planningAvecPertes(), acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}

	if c.bruts != 5 {
		t.Errorf("bruts = %d, want 5 (le total rendu par la source)", c.bruts)
	}

	parRaison := map[string]int{}
	for _, r := range c.rejets {
		parRaison[r.Raison]++
	}
	for raison, want := range map[string]int{
		rejetCocleVide: 2, // "" et "0"
		rejetPlcleVide: 1,
		rejetDoublonPL: 1,
	} {
		if parRaison[raison] != want {
			t.Errorf("rejets[%s] = %d, want %d (tous: %v)", raison, parRaison[raison], want, parRaison)
		}
	}

	// L'invariant du dump : chaque créneau rendu par la source est soit retenu,
	// soit rejeté, jamais les deux ni ni l'un ni l'autre. C'est lui qui rend le
	// fichier exploitable — un créneau manquant des deux côtés serait
	// précisément la perte silencieuse qu'on cherche à débusquer.
	//
	// Le doublon ne fausse pas le compte : c'est le créneau ÉVINCÉ qui est
	// consigné, l'évinçant prenant sa place parmi les retenus.
	if len(c.creneaux)+len(c.rejets) != c.bruts {
		t.Errorf("retenus(%d) + rejets(%d) ≠ bruts(%d)", len(c.creneaux), len(c.rejets), c.bruts)
	}
}

func TestTracerDesactiveNAccumulePasDeRejets(t *testing.T) {
	c, err := collecter(planningAvecPertes(), acTest(), true, false)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}
	if len(c.rejets) != 0 {
		t.Errorf("rejets = %d, want 0 : sans dump configuré, rien ne doit être mémorisé", len(c.rejets))
	}
	// Les écartements doivent tout de même avoir lieu.
	if _, vu := c.creneaux["x1"]; vu {
		t.Error("un créneau sans COCLE doit être écarté, dump ou pas")
	}
}

func TestEcrireDump(t *testing.T) {
	c, err := collecter(planningAvecPertes(), acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}

	chemin := filepath.Join(t.TempDir(), "planning.json")
	c.ecrireDump(chemin, acTest(), true)

	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("dump non écrit: %v", err)
	}
	var doc dumpCycle
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("dump illisible: %v", err)
	}

	if doc.Resume.CreneauxBruts != 5 || doc.Resume.CreneauxRetenus != len(c.creneaux) {
		t.Errorf("résumé incohérent: %+v", doc.Resume)
	}
	if !doc.Annee.Trouvee || doc.Annee.Annee != 2025 {
		t.Errorf("année = %+v, want 2025 trouvée", doc.Annee)
	}
	if len(doc.Promos) != 2 {
		t.Fatalf("promos = %d, want 2", len(doc.Promos))
	}
	for _, p := range doc.Promos {
		if !p.PlanningOK {
			t.Errorf("promo %s : planning_ok=false alors que le flux a répondu", p.P0cle)
		}
	}

	// Le tri doit être stable : deux dumps du même état se comparent au diff.
	for i := 1; i < len(doc.Creneaux); i++ {
		if doc.Creneaux[i-1].Date > doc.Creneaux[i].Date {
			t.Fatalf("créneaux non triés par date: %v", doc.Creneaux)
		}
	}
}

func TestEcrireDumpDesactiveNeCreeRien(t *testing.T) {
	dir := t.TempDir()
	c := &collecte{}
	c.ecrireDump("", acTest(), true)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("chemin vide : aucun fichier ne doit être écrit, trouvé %v", entries)
	}
}

// Le répertoire cible est créé au besoin : demander à l'exploitant de le créer
// à la main avant d'activer un outil de diagnostic serait une chausse-trappe.
func TestEcrireDumpCreeLeRepertoire(t *testing.T) {
	c, err := collecter(srcTest(), acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}
	chemin := filepath.Join(t.TempDir(), "a", "b", "planning.json")
	c.ecrireDump(chemin, acTest(), true)

	if _, err := os.Stat(chemin); err != nil {
		t.Fatalf("le dump doit créer son arborescence: %v", err)
	}
}

// Un chemin non inscriptible ne doit jamais faire échouer le cycle : le
// diagnostic est subordonné à la synchronisation, pas l'inverse. C'est le cas
// réel d'un chemin sous /var/lib alors que le service tourne en non-root.
func TestEcrireDumpCheminInterditNeFaitPasEchouer(t *testing.T) {
	c, err := collecter(srcTest(), acTest(), true, true)
	if err != nil {
		t.Fatalf("collecter: %v", err)
	}
	interdit := filepath.Join(t.TempDir(), "verrouille")
	if err := os.Mkdir(interdit, 0o500); err != nil {
		t.Fatal(err)
	}
	c.ecrireDump(filepath.Join(interdit, "sous", "planning.json"), acTest(), true)
}
