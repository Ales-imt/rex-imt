package webdfd

import (
	"strings"
	"testing"
)

func TestParseListeGroupe(t *testing.T) {
	t.Run("tabulation simple", func(t *testing.T) {
		// Encodage ASCII (sous-ensemble de Windows-1252) pour ce test.
		raw := []byte(strings.Join([]string{
			"Promotion :\t1A INFRES\tGroupe :\tINFRES\tMatière :\t\tContrôle :\t\tCoefficient :",
			"Num étudiant\tNom\tPrénom\tnote",
			"19475\tAHMED ALI\tAdoita\t",
			"19476\tALQUIER\tAurélien\t",
			"19477\tARROYO\tMaxime\t",
			"",
		}, "\n"))

		evs, groupeLabel, promoLabel, err := parseListeGroupe(raw)

		if err != nil {
			t.Fatalf("parseListeGroupe: %v", err)
		}

		if promoLabel != "1A INFRES" {
			t.Errorf("promoLabel = %q, want %q", promoLabel, "1A INFRES")
		}
		if groupeLabel != "INFRES" {
			t.Errorf("groupeLabel = %q, want %q", groupeLabel, "INFRES")
		}
		if len(evs) != 3 {
			t.Fatalf("evs: got %d, want 3", len(evs))
		}
		want := []string{"19475", "19476", "19477"}
		for i, w := range want {
			if evs[i] != w {
				t.Errorf("evs[%d] = %q, want %q", i, evs[i], w)
			}
		}
	})

	t.Run("nom composé n'est pas découpé", func(t *testing.T) {
		raw := []byte("Promotion :\t1A\tGroupe :\tG1\n" +
			"Num étudiant\tNom\tPrénom\tnote\n" +
			"10001\tAHMED ALI\tJean\t\n")
		evs, _, _, err := parseListeGroupe(raw)
		if err != nil {
			t.Fatalf("parseListeGroupe: %v", err)
		}
		if len(evs) != 1 || evs[0] != "10001" {
			t.Errorf("evs = %v, want [10001]", evs)
		}
	})

	t.Run("ligne note ignorée (préambule sauté)", func(t *testing.T) {
		raw := []byte("Promotion :\tPROMO\tGroupe :\tGR\n" +
			"Num étudiant\tNom\tPrénom\tnote\n" +
			"99999\tDUPONT\tPierre\t18.5\n")
		evs, _, _, err := parseListeGroupe(raw)
		if err != nil {
			t.Fatalf("parseListeGroupe: %v", err)
		}
		// La note est en 4e colonne : elle ne doit pas interférer avec l'EV.
		if len(evs) != 1 || evs[0] != "99999" {
			t.Errorf("evs = %v, want [99999]", evs)
		}
	})

	t.Run("clés vides et lignes non numériques ignorées", func(t *testing.T) {
		raw := []byte("Promotion :\tPROMO\tGroupe :\tGR\n" +
			"Num étudiant\tNom\tPrénom\tnote\n" +
			"\t\t\t\n" +
			"abc\tFOO\tBAR\t\n" +
			"20000\tMARTIN\tLuc\t\n")
		evs, _, _, err := parseListeGroupe(raw)
		if err != nil {
			t.Fatalf("parseListeGroupe: %v", err)
		}
		if len(evs) != 1 || evs[0] != "20000" {
			t.Errorf("evs = %v, want [20000]", evs)
		}
	})

	t.Run("groupe réellement vide : préambule seul, sans erreur", func(t *testing.T) {
		// Un groupe dépeuplé en amont conserve son préambule. C'est ce qui le
		// rend distinguable d'une page d'erreur, et donc réconciliable.
		raw := []byte("Promotion :\tPROMO\tGroupe :\tGR\n" +
			"Num étudiant\tNom\tPrénom\tnote\n")
		evs, groupeLabel, _, err := parseListeGroupe(raw)
		if err != nil {
			t.Fatalf("un groupe vide mais identifiable ne doit pas être une erreur: %v", err)
		}
		if len(evs) != 0 {
			t.Errorf("evs = %v, want []", evs)
		}
		if groupeLabel != "GR" {
			t.Errorf("groupeLabel = %q, want %q", groupeLabel, "GR")
		}
	})

	t.Run("flux vide : erreur", func(t *testing.T) {
		if _, _, _, err := parseListeGroupe([]byte{}); err == nil {
			t.Fatal("un flux vide doit être une erreur, pas un groupe vide")
		}
	})

	t.Run("page d'erreur amont : erreur", func(t *testing.T) {
		// cgiempt sert ses pages d'erreur en HTTP 200. Rendues comme un groupe
		// vide, elles videraient l'effectif attendu de toutes ses séances.
		raw := []byte("<html><head><title>Erreur</title></head>\n" +
			"<body>Session expirée, veuillez vous reconnecter.</body></html>\n")
		if _, _, _, err := parseListeGroupe(raw); err == nil {
			t.Fatal("une page d'erreur HTML doit être une erreur, pas un groupe vide")
		}
	})
}

func TestDetectDelimiter(t *testing.T) {
	if detectDelimiter([]byte("a\tb\tc")) != '\t' {
		t.Error("tabulation non détectée")
	}
	if detectDelimiter([]byte("a;b;c")) != ';' {
		t.Error("point-virgule non détecté en fallback")
	}
}

func TestIsNumeric(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"19475", true},
		{"0", true},
		{"", false},
		{"abc", false},
		{"12ab", false},
		{"12 34", false},
	}
	for _, tc := range cases {
		if got := isNumeric(tc.s); got != tc.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}
