package hfsqlexport

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// creerExport fabrique un répertoire d'export, avec ou sans son marqueur de fin.
func creerExport(t *testing.T, data, nom string, termine bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(data, nom, SousDossierJSON), 0o755); err != nil {
		t.Fatal(err)
	}
	if termine {
		if err := Marquer(data, nom); err != nil {
			t.Fatal(err)
		}
	}
}

// L'ordre lexicographique des noms doit être l'ordre chronologique : c'est ce
// qui permet de trier sans interroger le système de fichiers.
func TestNomExportOrdonnable(t *testing.T) {
	tot := NomExport(time.Date(2026, 3, 9, 8, 0, 0, 0, time.UTC))
	tard := NomExport(time.Date(2026, 3, 9, 14, 30, 0, 0, time.UTC))
	lendemain := NomExport(time.Date(2026, 3, 10, 1, 0, 0, 0, time.UTC))

	if !(tot < tard && tard < lendemain) {
		t.Fatalf("ordre non chronologique : %s %s %s", tot, tard, lendemain)
	}
}

func TestDernierPrendLePlusRecentComplet(t *testing.T) {
	data := t.TempDir()
	creerExport(t, data, "export-20250101-120000", true)
	creerExport(t, data, "export-20250703-093000", true)
	creerExport(t, data, "export-20250212-235959", true)
	// Un répertoire étranger ne doit pas être confondu avec un export.
	if err := os.MkdirAll(filepath.Join(data, "20250707.bd"), 0o755); err != nil {
		t.Fatal(err)
	}

	nom, jsonDir, err := Dernier(data)
	if err != nil {
		t.Fatal(err)
	}
	if nom != "export-20250703-093000" {
		t.Fatalf("nom %q", nom)
	}
	if jsonDir != filepath.Join(data, nom, SousDossierJSON) {
		t.Fatalf("jsonDir %q", jsonDir)
	}
}

// C'est LA garantie du contrat : un export en cours d'écriture n'a pas encore
// son marqueur, et doit rester invisible. Sinon le consommateur lirait un
// planning tronqué et annulerait des séances à tort.
func TestDernierIgnoreExportSansMarqueur(t *testing.T) {
	data := t.TempDir()
	creerExport(t, data, "export-20250101-120000", true)
	creerExport(t, data, "export-20250703-093000", false) // en cours

	nom, _, err := Dernier(data)
	if err != nil {
		t.Fatal(err)
	}
	if nom != "export-20250101-120000" {
		t.Fatalf("nom %q : l'export incomplet a été retenu", nom)
	}
}

func TestDernierAucunExport(t *testing.T) {
	nom, jsonDir, err := Dernier(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if nom != "" || jsonDir != "" {
		t.Fatalf("nom %q, jsonDir %q, attendus vides", nom, jsonDir)
	}
}

// Répertoire absent : erreur remontée, pas un silence. Le consommateur en fera
// ce qu'il veut (le producteur n'a peut-être pas encore démarré), mais il doit
// pouvoir distinguer « rien à lire » de « je ne sais pas lire ».
func TestDernierRepertoireAbsent(t *testing.T) {
	if _, _, err := Dernier(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("répertoire absent accepté")
	}
}

func TestPurgerConserveLesPlusRecents(t *testing.T) {
	data := t.TempDir()
	for _, nom := range []string{
		"export-20250101-120000", "export-20250212-235959",
		"export-20250703-093000", "export-20250704-093000",
	} {
		creerExport(t, data, nom, true)
	}
	if err := os.MkdirAll(filepath.Join(data, "20250707.bd"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Purger(data, 2); err != nil {
		t.Fatal(err)
	}

	restants, err := Lister(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"export-20250703-093000", "export-20250704-093000"}
	if len(restants) != len(want) || restants[0] != want[0] || restants[1] != want[1] {
		t.Fatalf("exports restants %v, attendu %v", restants, want)
	}
	// La base dézippée n'est pas un export : elle ne doit pas être emportée.
	if _, err := os.Stat(filepath.Join(data, "20250707.bd")); err != nil {
		t.Fatalf("répertoire étranger supprimé : %v", err)
	}
}

func TestPurgerDesactivee(t *testing.T) {
	data := t.TempDir()
	creerExport(t, data, "export-20250101-120000", true)
	creerExport(t, data, "export-20250703-093000", true)

	if err := Purger(data, 0); err != nil {
		t.Fatal(err)
	}
	restants, err := Lister(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(restants) != 2 {
		t.Fatalf("%d exports restants, attendu 2", len(restants))
	}
}
