package webdfd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// serveur rend une source branchée sur un serveur qui répond toujours le même
// corps, quels que soient les paramètres — ce qui suffit : chaque flux est
// interrogé par une méthode distincte.
func serveur(t *testing.T, corps string) *Source {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, corps)
	}))
	t.Cleanup(srv.Close)
	return &Source{BaseURL: srv.URL, ListeGroupeURL: srv.URL}
}

func TestPromos(t *testing.T) {
	src := serveur(t, "P0;94;NOM;1A BAT 2026-27;COLORI;1;CFOND;10987431;CTEXTE;0\n"+
		"P0;95;NOM;2A BAT 2025-26;COLORI;2;CFOND;10987432;CTEXTE;1\n"+
		"P0;;NOM;sans identifiant\n"+
		"P0;abc;NOM;identifiant non numérique\n"+
		"P0;96;NOM;\n"+
		"\n"+
		"EOT\n")

	promos, err := src.Promos()
	if err != nil {
		t.Fatal(err)
	}
	if len(promos) != 2 {
		t.Fatalf("%d promotions, attendu 2 : %+v", len(promos), promos)
	}
	if promos[0].ExternalID != "94" || promos[0].Nom != "1A BAT 2026-27" {
		t.Errorf("promotion %+v", promos[0])
	}
	if promos[1].ExternalID != "95" || promos[1].Nom != "2A BAT 2025-26" {
		t.Errorf("promotion %+v", promos[1])
	}
}

func TestProfsEtEleves(t *testing.T) {
	// Corps encodé en Windows-1252, comme cybema le sert : "\xe9" est le é. Sans
	// le décodage, il ressortirait en « Ã© » côté base.
	profs, err := serveur(t, "PR;248;DET;M. ;NOM;LECOEUCHE ;PRENOM;St\xe9phane ;MEL;Stephane.LECOEUCHE@mines-ales.fr \nEOT\n").Profs()
	if err != nil {
		t.Fatal(err)
	}
	if len(profs) != 1 {
		t.Fatalf("%d profs, attendu 1", len(profs))
	}
	if profs[0].ExternalID != "248" || profs[0].Nom != "LECOEUCHE" || profs[0].Prenom != "Stéphane" {
		t.Errorf("prof %+v", profs[0])
	}
	if profs[0].Email != "stephane.lecoeuche@mines-ales.fr" {
		t.Errorf("email %q, attendu en minuscules", profs[0].Email)
	}

	eleves, err := serveur(t, "EV;18467;DET;Mme ;NOM;ABBAS ;PRENOM;Louna ;MEL;louna.abbas@etu.mines-ales.fr ;P0;2 \nEOT\n").Eleves()
	if err != nil {
		t.Fatal(err)
	}
	if len(eleves) != 1 || eleves[0].ExternalID != "18467" || eleves[0].Nom != "ABBAS" {
		t.Fatalf("élèves %+v", eleves)
	}
}

func TestPlanning(t *testing.T) {
	// Ligne réelle du flux, tronquée après les champs consommés. NOTE est la
	// CLÉ de la note amont, LANOTE son texte ("R\xe9union" : Windows-1252).
	src := serveur(t, "PL;20052;P0CLE;95;PRCLE;345;COCLE;2108;GRCLE;1734;SACLE;2;DATE;20260901;"+
		"HD;0800;HF;1000;TYPE; ;COURS;DPPA ;SALLE;A - BAUJON - CLAV ;PROMO;1A MKX 2026-27 ;"+
		"PROF;M. VIELJUS ;GROUPE;- ;NOTE;652288;LANOTE;R\xe9union DRDV \nEOT\n")

	entries, err := src.Planning("95", "20260801", "20260930")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("%d créneaux, attendu 1", len(entries))
	}
	e := entries[0]
	if e.Plcle != "20052" || e.P0cle != "95" || e.Cocle != "2108" || e.Grcle != "1734" || e.Prcle != "345" {
		t.Errorf("clés %+v", e)
	}
	if e.Date != "20260901" || e.HD != "0800" || e.HF != "1000" {
		t.Errorf("horaires %+v", e)
	}
	// Format du champ PROF : civilité + NOM, sans prénom. C'est ce que la source
	// hfsql doit reproduire (cf. watcher.Source).
	if e.Prof != "M. VIELJUS" {
		t.Errorf("prof %q", e.Prof)
	}
	if e.Cours != "DPPA" || e.Salle != "A - BAUJON - CLAV" || e.Groupe != "-" {
		t.Errorf("libellés %+v", e)
	}
	// La note vient de LANOTE, jamais de NOTE (qui n'est que sa clé).
	if e.Note != "Réunion DRDV" {
		t.Errorf("note %q", e.Note)
	}
}

func TestPlanningIgnoreLignesSansDate(t *testing.T) {
	entries, err := serveur(t, "PL;1;P0CLE;2;COURS;sans date\nEOT\n").Planning("2", "20240101", "20261231")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("%d créneaux, attendu 0", len(entries))
	}
}

func TestErreurHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := (&Source{BaseURL: srv.URL}).Promos(); err == nil {
		t.Fatal("HTTP 500 accepté")
	}
}

// TestTimeout vérifie qu'une requête est bornée dans le temps. L'enjeu n'est
// pas la valeur du délai mais le fait que les appels passent bien par `client`
// : avec http.Get, un amont qui accepte la connexion sans répondre bloquerait
// le cycle indéfiniment, et rien dans les tests ne le signalerait.
func TestTimeout(t *testing.T) {
	precedent := client
	client = &http.Client{Timeout: 20 * time.Millisecond}
	t.Cleanup(func() { client = precedent })

	// Le serveur répond, mais trop tard. La tempo reste courte : srv.Close
	// attend les requêtes en cours.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	debut := time.Now()
	if _, err := (&Source{BaseURL: srv.URL}).Promos(); err == nil {
		t.Fatal("requête sans réponse acceptée")
	}
	if ecoule := time.Since(debut); ecoule > 250*time.Millisecond {
		t.Fatalf("requête interrompue au bout de %s, le timeout n'a pas joué", ecoule)
	}
}

func TestParseKV(t *testing.T) {
	cases := []struct {
		line string
		want map[string]string
	}{
		{
			line: "P0;94;NOM;1A BAT 2026-27;COLORI;1;CFOND;10987431;CTEXTE;0",
			want: map[string]string{
				"P0": "94", "NOM": "1A BAT 2026-27",
				"COLORI": "1", "CFOND": "10987431", "CTEXTE": "0",
			},
		},
		{
			line: "P0;1;NOM;TEST ",
			want: map[string]string{"P0": "1", "NOM": "TEST"},
		},
		{
			line: "cle_seule",
			want: map[string]string{},
		},
	}

	for _, tc := range cases {
		got := parseKV(tc.line)
		for k, v := range tc.want {
			if got[k] != v {
				t.Errorf("parseKV(%q)[%q] = %q, want %q", tc.line, k, got[k], v)
			}
		}
		if len(got) != len(tc.want) {
			t.Errorf("parseKV(%q): got %d clefs, want %d", tc.line, len(got), len(tc.want))
		}
	}
}
