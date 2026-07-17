package ledger_test

import (
	"back-rex-common/pkg/ledger"
	"testing"
	"time"
)

var baseEntry = ledger.LedgerEntry{
	SeanceID:   42,
	UserID:     7,
	Statut:     "PRESENT",
	EventAt:    time.Date(2026, 6, 29, 14, 30, 0, 123456789, time.UTC),
	RecordedAt: time.Date(2026, 6, 29, 14, 30, 0, 987654321, time.UTC),
	PrevHash:   ledger.GenesisHash,
}

func TestComputeHash_Determinism(t *testing.T) {
	h1 := ledger.ComputeHash(baseEntry)
	h2 := ledger.ComputeHash(baseEntry)
	if h1 != h2 {
		t.Fatalf("non-deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("hash length %d, want 64", len(h1))
	}
}

func TestComputeHash_Sensitivity(t *testing.T) {
	ref := ledger.ComputeHash(baseEntry)

	cases := []struct {
		name  string
		entry ledger.LedgerEntry
	}{
		{"SeanceID changed", func() ledger.LedgerEntry { e := baseEntry; e.SeanceID = 43; return e }()},
		{"UserID changed", func() ledger.LedgerEntry { e := baseEntry; e.UserID = 8; return e }()},
		{"Statut changed", func() ledger.LedgerEntry { e := baseEntry; e.Statut = "RETARD"; return e }()},
		{"EventAt changed", func() ledger.LedgerEntry {
			e := baseEntry
			e.EventAt = baseEntry.EventAt.Add(time.Microsecond)
			return e
		}()},
		{"RecordedAt changed", func() ledger.LedgerEntry {
			e := baseEntry
			e.RecordedAt = baseEntry.RecordedAt.Add(time.Microsecond)
			return e
		}()},
		{"PrevHash changed", func() ledger.LedgerEntry {
			e := baseEntry
			e.PrevHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			return e
		}()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := ledger.ComputeHash(tc.entry)
			if h == ref {
				t.Errorf("hash unchanged after mutation %q", tc.name)
			}
		})
	}
}

func TestComputeHash_FieldOrderStability(t *testing.T) {
	// Two entries that differ only in field order (impossible in Go struct,
	// but we test that canonical serialisation is stable across calls).
	e1 := baseEntry
	e2 := baseEntry
	if ledger.ComputeHash(e1) != ledger.ComputeHash(e2) {
		t.Fatal("hash is not stable")
	}
}

// TestComputeHash_DatabaseRoundTrip verrouille le contrat de précision : le
// hash calculé à l'insertion (time.Now() nanoseconde) doit être reproductible
// après relecture depuis PostgreSQL, qui tronque timestamptz à la microseconde
// (pgx encode Nanosecond()/1000). Sans troncature dans ComputeHash, toute
// vérification de chaîne échouerait dès le premier maillon.
func TestComputeHash_DatabaseRoundTrip(t *testing.T) {
	atInsert := baseEntry // EventAt/RecordedAt portent des nanosecondes (…789, …321)

	afterRoundTrip := baseEntry
	afterRoundTrip.EventAt = baseEntry.EventAt.Truncate(time.Microsecond)
	afterRoundTrip.RecordedAt = baseEntry.RecordedAt.Truncate(time.Microsecond)

	if h1, h2 := ledger.ComputeHash(atInsert), ledger.ComputeHash(afterRoundTrip); h1 != h2 {
		t.Fatalf("hash not reproducible after microsecond round-trip: %q != %q", h1, h2)
	}
}

func TestComputeHash_GenesisIsRecognisable(t *testing.T) {
	if len(ledger.GenesisHash) != 64 {
		t.Fatalf("GenesisHash length %d, want 64", len(ledger.GenesisHash))
	}
	for _, c := range ledger.GenesisHash {
		if c != '0' {
			t.Fatalf("GenesisHash contains non-zero character: %q", ledger.GenesisHash)
		}
	}
}
