package migration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSyncMatieres(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "CO;1456;NOM;RATTRAPAGE ;COLORI;0;CFOND;16777215;CTEXTE;0;P0;0\n")
		fmt.Fprint(w, "CO;1457;NOM;9.2 MATHEMATIQUES;COLORI;1;CFOND;16777215;CTEXTE;0;P0;94\n")
		fmt.Fprint(w, "\n")
		fmt.Fprint(w, "EOT\n")
	}))
	defer srv.Close()

	ctx := context.Background()
	db, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	if err = db.Ping(ctx); err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	defer db.Close()

	// Nettoyage : la FK ON DELETE CASCADE propage sur migration.matiere_map.
	db.Exec(ctx, `
		DELETE FROM public.matiere m
		USING migration.matiere_map mm
		WHERE mm.internal_id = m.id AND mm.source = 'webdfd' AND mm.external_id IN ('1456','1457')`)

	if err = SyncMatieres(ctx, srv.URL, db); err != nil {
		t.Fatalf("SyncMatieres: %v", err)
	}

	t.Run("matiere.name", func(t *testing.T) {
		rows, err := db.Query(ctx, `
			SELECT m.name FROM public.matiere m
			JOIN migration.matiere_map mm ON mm.internal_id = m.id
			WHERE mm.source = 'webdfd' AND mm.external_id IN ('1456','1457')
			ORDER BY mm.external_id::integer`)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()

		want := []string{"RATTRAPAGE", "9.2 MATHEMATIQUES"}
		var got []string
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got = append(got, name)
		}
		if len(got) != len(want) {
			t.Fatalf("matieres: got %d, want %d", len(got), len(want))
		}
		for i, w := range want {
			if got[i] != w {
				t.Errorf("matiere[%d]: got %q, want %q", i, got[i], w)
			}
		}
	})

	t.Run("matiere_map.external_id", func(t *testing.T) {
		rows, err := db.Query(ctx,
			`SELECT external_id FROM migration.matiere_map
			 WHERE source = 'webdfd' AND external_id IN ('1456','1457')
			 ORDER BY external_id::integer`)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()

		var got []string
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got = append(got, id)
		}
		if len(got) != 2 {
			t.Fatalf("matiere_map: got %d entrées, want 2", len(got))
		}
		if got[0] != "1456" || got[1] != "1457" {
			t.Errorf("matiere_map: external_ids = %v, want [1456 1457]", got)
		}
	})
}

func TestSemesterFromName(t *testing.T) {
	cases := []struct {
		nom      string
		wantName string
		wantOK   bool
	}{
		{"9.2.1 PROJET INTÉGRATEUR", "S9", true},
		{"1.1 ANGLAIS", "S1", true},
		{"RATTRAPAGE", "INCONNU", false},
		{"0.1 COURS", "INCONNU", false},
		{"", "INCONNU", false},
		{"abc.def NOM", "INCONNU", false},
	}
	for _, tc := range cases {
		got, ok := semesterFromName(tc.nom)
		if ok != tc.wantOK || got != tc.wantName {
			t.Errorf("semesterFromName(%q) = (%q, %v), want (%q, %v)",
				tc.nom, got, ok, tc.wantName, tc.wantOK)
		}
	}
}
