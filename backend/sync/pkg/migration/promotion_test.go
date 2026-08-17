package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testDSN = "host=10.20.1.4 port=5432 user=postgres password=root dbname=db_rex sslmode=disable"

func TestSyncPromotions(t *testing.T) {
	src := sourceFixe{promos: []source.Promo{
		{ExternalID: "94", Nom: "1A BAT 2026-27"},
		{ExternalID: "95", Nom: "2A BAT 2025-26"},
	}}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	if err = db.Ping(ctx); err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	defer db.Close()

	// Nettoyage : la FK ON DELETE CASCADE propage sur migration.promotion_map.
	db.Exec(ctx, "DELETE FROM public.promotion WHERE name = ANY($1)",
		[]string{"1A BAT 2026-27", "2A BAT 2025-26"})

	ac, ok, err := getAnneeCourante(ctx, New(db), time.Now())
	if err != nil {
		t.Fatalf("getAnneeCourante: %v", err)
	}
	if !ok {
		t.Skip("aucune année ne correspond à la date du jour")
	}

	if err = syncPromotions(ctx, New(db), src.promos, ac); err != nil {
		t.Fatalf("syncPromotions: %v", err)
	}

	// Récupère les IDs internes créés.
	type promoInfo struct {
		id   int64
		name string
	}
	var promos []promoInfo
	{
		rows, err := db.Query(ctx,
			"SELECT id, name FROM public.promotion WHERE name = ANY($1) ORDER BY id",
			[]string{"1A BAT 2026-27", "2A BAT 2025-26"})
		if err != nil {
			t.Fatalf("fetch promo ids: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			var p promoInfo
			if err = rows.Scan(&p.id, &p.name); err != nil {
				t.Fatalf("scan promo: %v", err)
			}
			promos = append(promos, p)
		}
		if len(promos) != 2 {
			t.Fatalf("promotions créées: got %d, want 2", len(promos))
		}
	}

	t.Run("promotion.name", func(t *testing.T) {
		want := map[int64]string{}
		for _, p := range promos {
			want[p.id] = p.name
		}
		rows, err := db.Query(ctx,
			"SELECT id, name FROM public.promotion WHERE id = ANY($1) ORDER BY id",
			[]int64{promos[0].id, promos[1].id})
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()
		got := map[int64]string{}
		for rows.Next() {
			var id int64
			var name string
			if err = rows.Scan(&id, &name); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got[id] = name
		}
		for id, name := range want {
			if got[id] != name {
				t.Errorf("promotion id=%d: got %q, want %q", id, got[id], name)
			}
		}
	})

	t.Run("promotion_map.webdfd", func(t *testing.T) {
		rows, err := db.Query(ctx,
			`SELECT internal_id, external_id FROM migration.promotion_map
			 WHERE source = 'webdfd' AND internal_id = ANY($1) ORDER BY external_id::integer`,
			[]int64{promos[0].id, promos[1].id})
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()

		type entry struct {
			internalID int64
			externalID string
		}
		var got []entry
		for rows.Next() {
			var e entry
			if err = rows.Scan(&e.internalID, &e.externalID); err != nil {
				t.Fatalf("scan: %v", err)
			}
			got = append(got, e)
		}
		if len(got) != 2 {
			t.Fatalf("promotion_map: got %d entrées, want 2", len(got))
		}
		if got[0].externalID != "94" || got[1].externalID != "95" {
			t.Errorf("promotion_map: external_ids = [%s %s], want [94 95]",
				got[0].externalID, got[1].externalID)
		}
	})
}
