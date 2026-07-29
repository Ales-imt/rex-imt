package migration

import (
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestFirstSeanceByCocle(t *testing.T) {
	seances := map[string]planningEntry{
		"pl1": {cocle: "10", date: "20250915", hd: "0800"},
		"pl2": {cocle: "10", date: "20250908", hd: "1000"}, // plus tôt → gagnant
		"pl3": {cocle: "10", date: "20251201", hd: "0800"},
		"pl4": {cocle: "20", date: "20260210", hd: "1400"},
		"pl5": {cocle: "0", date: "20250908", hd: "0800"}, // COCLE fictif ignoré
		"pl6": {cocle: "30", date: "bad", hd: "0800"},     // date illisible ignorée
	}

	got := firstSeanceByCocle(seances)

	if len(got) != 2 {
		t.Fatalf("firstSeanceByCocle: %d matières, want 2 (%v)", len(got), got)
	}
	want10 := time.Date(2025, 9, 8, 10, 0, 0, 0, parisLoc)
	if !got["10"].Equal(want10) {
		t.Errorf("cocle 10: got %v, want %v", got["10"], want10)
	}
	if _, ok := got["30"]; ok {
		t.Errorf("cocle 30: attendu absent (date illisible)")
	}
}

func TestAnneeForDate(t *testing.T) {
	annees := []ListAnneesRow{
		{ID: 1, Debut: date(2024, 9, 1), Fin: date(2025, 8, 31)},
		{ID: 2, Debut: date(2025, 9, 1), Fin: date(2026, 8, 31)},
	}
	cases := []struct {
		name      string
		t         time.Time
		wantAnnee int32
		wantOK    bool
	}{
		{"début d'année scolaire", time.Date(2025, 9, 15, 8, 0, 0, 0, parisLoc), 2025, true},
		{"printemps même année", time.Date(2026, 2, 10, 14, 0, 0, 0, parisLoc), 2025, true},
		{"borne debut incluse", date(2025, 9, 1), 2025, true},
		{"borne fin incluse en soirée", time.Date(2025, 8, 31, 23, 0, 0, 0, parisLoc), 2024, true},
		{"année précédente", time.Date(2024, 10, 1, 8, 0, 0, 0, parisLoc), 2024, true},
		{"hors de toute année", time.Date(2027, 1, 1, 8, 0, 0, 0, parisLoc), 0, false},
	}
	for _, tc := range cases {
		got, ok := anneeForDate(annees, tc.t)
		if ok != tc.wantOK || got != tc.wantAnnee {
			t.Errorf("%s: anneeForDate(%v) = (%d, %v), want (%d, %v)",
				tc.name, tc.t, got, ok, tc.wantAnnee, tc.wantOK)
		}
	}
}
