package watcher

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeZip fabrique une archive contenant les entrees donnees (nom -> contenu,
// un nom terminant par / etant un repertoire).
func writeZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for entry, content := range entries {
		hw, err := w.Create(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := hw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestZipRoot(t *testing.T) {
	tests := []struct {
		nom     string
		entries map[string]string
		root    string
		erreur  string
	}{
		{
			nom:     "base hfsql",
			entries: map[string]string{"20250707.bd/eleves.FIC": "x", "20250707.bd/eleves.NDX": "y"},
			root:    "20250707.bd",
		},
		{
			nom:     "fichier a la racine",
			entries: map[string]string{"eleves.FIC": "x"},
			erreur:  "hors d'un repertoire racine",
		},
		{
			nom:     "plusieurs racines",
			entries: map[string]string{"a.bd/x": "1", "b.bd/y": "2"},
			erreur:  "plusieurs repertoires racine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			root, err := ZipRoot(writeZip(t, tt.entries))
			switch {
			case tt.erreur != "":
				if err == nil || !strings.Contains(err.Error(), tt.erreur) {
					t.Fatalf("erreur attendue contenant %q, obtenu %v", tt.erreur, err)
				}
			case err != nil:
				t.Fatalf("erreur inattendue : %v", err)
			case root != tt.root:
				t.Fatalf("racine %q, attendu %q", root, tt.root)
			}
		})
	}
}

func TestVerifyZipRejetteArchiveTronquee(t *testing.T) {
	name := writeZip(t, map[string]string{"20250707.bd/eleves.FIC": strings.Repeat("données", 500)})

	if err := VerifyZip(name); err != nil {
		t.Fatalf("archive valide rejetée : %v", err)
	}

	// Une copie encore en cours ressemble a une archive tronquee.
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	partiel := filepath.Join(t.TempDir(), "partiel.zip")
	contenu, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(partiel, contenu[:info.Size()/2], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyZip(partiel); err == nil {
		t.Fatal("archive tronquée acceptée")
	}
}

func TestUnzipRefuseZipSlip(t *testing.T) {
	name := writeZip(t, map[string]string{"../evade.txt": "x"})
	dest := t.TempDir()

	if err := Unzip(name, dest); err == nil {
		t.Fatal("entrée hors du répertoire cible acceptée")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "evade.txt")); err == nil {
		t.Fatal("fichier écrit hors du répertoire cible")
	}
}

func TestUnzipExtraitLaBase(t *testing.T) {
	fichiers := map[string]string{
		"20250707.bd/eleves.FIC": "données élèves",
		"20250707.bd/eleves.NDX": "index",
	}
	dest := t.TempDir()

	if err := Unzip(writeZip(t, fichiers), dest); err != nil {
		t.Fatal(err)
	}
	for entry, attendu := range fichiers {
		obtenu, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(entry)))
		if err != nil {
			t.Fatal(err)
		}
		if string(obtenu) != attendu {
			t.Fatalf("%s : contenu %q, attendu %q", entry, obtenu, attendu)
		}
	}
}
