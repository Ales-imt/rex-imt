package hfsql

import (
	"back-rex-sync/pkg/source"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exportTest = "testdata/export/json"

func monter(t *testing.T) *Source {
	t.Helper()
	s, err := NewSource(exportTest)
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	return s
}

// Les chaînes HFSQL sont stockées à largeur fixe : sans nettoyage, chaque nom
// arriverait en base suivi de vingt espaces.
func TestHFSQLPromos(t *testing.T) {
	promos, err := monter(t).Promos()
	if err != nil {
		t.Fatal(err)
	}
	// La promotion au nom vide n'en est pas une : elle est écartée, comme le
	// fait le parseur webdfd pour une ligne sans NOM.
	if len(promos) != 2 {
		t.Fatalf("%d promotions, attendu 2 : %+v", len(promos), promos)
	}
	if promos[0].ExternalID != "2" || promos[0].Nom != "1A FISE" {
		t.Errorf("promotion %+v, attendu {2 1A FISE}", promos[0])
	}
	if promos[1].Nom != "Réservation Formation continue" {
		t.Errorf("nom %q, accents attendus intacts", promos[1].Nom)
	}
}

func TestHFSQLProfs(t *testing.T) {
	profs, err := monter(t).Profs()
	if err != nil {
		t.Fatal(err)
	}
	if len(profs) != 2 {
		t.Fatalf("%d profs, attendu 2", len(profs))
	}
	if profs[0].ExternalID != "248" || profs[0].Nom != "LECOEUCHE" || profs[0].Prenom != "Stéphane" {
		t.Errorf("prof %+v", profs[0])
	}
	// L'email sert de clé de rattachement : il doit arriver en minuscules,
	// comme côté webdfd.
	if profs[0].Email != "stephane.lecoeuche@mines-ales.fr" {
		t.Errorf("email %q, attendu en minuscules", profs[0].Email)
	}
	// Un prof sans email est rendu tel quel : c'est SyncProfs qui décide de
	// l'ignorer, et qui le journalise.
	if profs[1].Email != "" {
		t.Errorf("email %q, attendu vide", profs[1].Email)
	}
}

func TestHFSQLEleves(t *testing.T) {
	eleves, err := monter(t).Eleves()
	if err != nil {
		t.Fatal(err)
	}
	if len(eleves) != 2 {
		t.Fatalf("%d élèves, attendu 2", len(eleves))
	}
	if eleves[0].ExternalID != "16219" || eleves[0].Nom != "FONTUGNE" {
		t.Errorf("élève %+v", eleves[0])
	}
	if eleves[0].Email != "gabin.fontugne@etu.mines-ales.fr" {
		t.Errorf("email %q, attendu en minuscules", eleves[0].Email)
	}
}

func TestHFSQLCoursNoms(t *testing.T) {
	noms, err := monter(t).CoursNoms()
	if err != nil {
		t.Fatal(err)
	}
	if noms["25"] != "9.2.1 PROJET" {
		t.Errorf("cours 25 = %q, attendu %q", noms["25"], "9.2.1 PROJET")
	}
	if len(noms) != 2 {
		t.Errorf("%d matières, attendu 2", len(noms))
	}
}

// L'export livre le planning de toutes les promotions et de toutes les années
// d'un bloc ; le filtrage que webdfd délègue au serveur se fait ici en mémoire.
func TestHFSQLPlanningFiltre(t *testing.T) {
	entries, err := monter(t).Planning("2", "20240801", "20250131")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]source.Creneau{}
	for _, e := range entries {
		got[e.Plcle] = e
	}
	if len(got) != 2 {
		t.Fatalf("%d créneaux, attendu 2 (PL 4 et 8) : %+v", len(got), got)
	}
	for _, exclu := range []string{"12", "13", "20"} {
		if _, ok := got[exclu]; ok {
			t.Errorf("PL=%s retenu alors qu'il est hors promotion ou hors plage", exclu)
		}
	}
}

// L'export ne porte que des clés là où webdfd livrait aussi les libellés :
// salle, prof, matière et groupe doivent être résolus par jointure.
func TestHFSQLPlanningJointures(t *testing.T) {
	entries, err := monter(t).Planning("2", "20240801", "20250131")
	if err != nil {
		t.Fatal(err)
	}
	var pl4, pl8 source.Creneau
	for _, e := range entries {
		switch e.Plcle {
		case "4":
			pl4 = e
		case "8":
			pl8 = e
		}
	}

	if pl4.Date != "20240904" || pl4.HD != "0800" || pl4.HF != "1030" {
		t.Errorf("horaires %+v", pl4)
	}
	if pl4.P0cle != "2" || pl4.Cocle != "25" || pl4.Grcle != "1074" || pl4.Prcle != "248" {
		t.Errorf("clés %+v", pl4)
	}
	if pl4.Salle != "A - PEYRE - CLAV" {
		t.Errorf("salle %q", pl4.Salle)
	}
	// Format du champ PROF de webdfd : civilité + NOM, sans prénom.
	if pl4.Prof != "M. LECOEUCHE" {
		t.Errorf("prof %q, attendu %q", pl4.Prof, "M. LECOEUCHE")
	}
	if pl4.Cours != "9.2.1 PROJET" {
		t.Errorf("cours %q", pl4.Cours)
	}
	if pl4.Groupe != "3/3" {
		t.Errorf("groupe %q", pl4.Groupe)
	}

	// Une clé à 0 vaut « pas de valeur » : le libellé doit rester vide plutôt
	// que de désigner la salle ou le prof numéro zéro.
	if pl8.Salle != "" || pl8.Prof != "" {
		t.Errorf("PL=8 : salle %q / prof %q, attendus vides", pl8.Salle, pl8.Prof)
	}
}

func TestHFSQLMembresGroupe(t *testing.T) {
	s := monter(t)

	evs, libelle, err := s.MembresGroupe("471")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 || evs[0] != "396" || evs[1] != "16219" {
		t.Errorf("membres %v, attendu [396 16219]", evs)
	}
	// L'export ne porte pas le préambule que webdfd sert avec sa liste : pas de
	// libellé, donc pas de garde-fou de cohérence côté SyncElevesGroupe.
	if libelle != "" {
		t.Errorf("libellé %q, attendu vide", libelle)
	}

	// Un groupe sans membre n'est pas une erreur : l'instantané est complet,
	// une absence signifie bien un groupe vide.
	evs, _, err = s.MembresGroupe("9999")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 0 {
		t.Errorf("membres %v, attendu aucun", evs)
	}
}

// Un export incomplet doit se signaler au montage, avant que la
// synchronisation n'écrive quoi que ce soit en base.
func TestNewSourceFichierManquant(t *testing.T) {
	dir := t.TempDir()
	copierExport(t, dir)
	if err := os.Remove(filepath.Join(dir, "planning.json")); err != nil {
		t.Fatal(err)
	}

	_, err := NewSource(dir)
	if err == nil {
		t.Fatal("export incomplet accepté")
	}
	if !strings.Contains(err.Error(), "planning.json") {
		t.Errorf("erreur %v, attendu un message nommant le fichier manquant", err)
	}
}

func TestNewSourceJSONInvalide(t *testing.T) {
	dir := t.TempDir()
	copierExport(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "promos.json"), []byte("{ pas du json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewSource(dir); err == nil {
		t.Fatal("JSON invalide accepté")
	}
}

// lireTable prend l'unique valeur de l'objet racine plutôt que d'entretenir la
// liste des noms de tables ; encore faut-il qu'elle soit unique.
func TestLireTablePlusieursRacines(t *testing.T) {
	dir := t.TempDir()
	name := "ambigu.json"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(`{"a":[],"b":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var dest []hfPromo
	err := lireTable(dir, name, &dest)
	if err == nil {
		t.Fatal("objet à deux racines accepté")
	}
	if !strings.Contains(err.Error(), "une seule attendue") {
		t.Errorf("erreur %v", err)
	}
}

// copierExport recopie l'export de référence dans dir, pour pouvoir l'abîmer
// sans toucher aux fixtures.
func copierExport(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(exportTest)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(exportTest, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
