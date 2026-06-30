package presence_test

import (
	"back-rex-common/pkg/ledger"
	"back-rex-common/pkg/services"
	"back-rex-admin/pkg/presence"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ── Stub DBTX ────────────────────────────────────────────────────────────────

// stubRow simulates pgx.Row.
type stubRow struct {
	values []any
	err    error
}

func (r *stubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if i >= len(r.values) {
			break
		}
		switch v := d.(type) {
		case *int64:
			*v = r.values[i].(int64)
		case *int32:
			*v = r.values[i].(int32)
		case *string:
			*v = r.values[i].(string)
		}
	}
	return nil
}

// stubRows simulates pgx.Rows backed by a slice.
type stubRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *stubRows) Next() bool        { r.idx++; return r.idx <= len(r.rows) }
func (r *stubRows) Close()            {}
func (r *stubRows) Err() error        { return r.err }
func (r *stubRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *stubRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *stubRows) Values() ([]any, error) { return nil, nil }
func (r *stubRows) RawValues() [][]byte { return nil }
func (r *stubRows) Conn() *pgx.Conn { return nil }
func (r *stubRows) Scan(dest ...any) error {
	if r.idx < 1 || r.idx > len(r.rows) {
		return errors.New("out of range")
	}
	row := r.rows[r.idx-1]
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		switch v := d.(type) {
		case *int64:
			*v = row[i].(int64)
		case *int32:
			*v = row[i].(int32)
		case *string:
			*v = row[i].(string)
		}
	}
	return nil
}

// stubDB implements the DBTX interface used by VerifyChain.
type stubDB struct {
	queryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	execFunc     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s *stubDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if s.queryFunc != nil {
		return s.queryFunc(ctx, sql, args...)
	}
	return &stubRows{}, nil
}
func (s *stubDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if s.queryRowFunc != nil {
		return s.queryRowFunc(ctx, sql, args...)
	}
	return &stubRow{err: pgx.ErrNoRows}
}
func (s *stubDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if s.execFunc != nil {
		return s.execFunc(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// buildChain builds n consecutive ledger entries in memory and returns their
// rows in the format expected by VerifyChain's Query scan.
// Row columns: seq, seance_id, user_id, statut, event_at, recorded_at, prev_hash, hash
func buildChainRows(n int) [][]any {
	t0 := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	prevHash := ledger.GenesisHash
	rows := make([][]any, 0, n)
	for i := 0; i < n; i++ {
		eventAt := t0.Add(time.Duration(i) * time.Minute)
		recordedAt := eventAt.Add(100 * time.Millisecond)
		e := ledger.LedgerEntry{
			SeanceID:   1,
			UserID:     int32(i + 1),
			Statut:     "PRESENT",
			EventAt:    eventAt,
			RecordedAt: recordedAt,
			PrevHash:   prevHash,
		}
		h := ledger.ComputeHash(e)
		rows = append(rows, []any{
			int64(i + 1), // seq
			int64(1),     // seance_id
			int32(i + 1), // user_id
			"PRESENT",    // statut
			eventAt,      // event_at  (stubRows.Scan handles time.Time via pgtype below)
			recordedAt,   // recorded_at
			prevHash,     // prev_hash
			h,            // hash
		})
		prevHash = h
	}
	return rows
}

// ── Tests VerifyChain ────────────────────────────────────────────────────────

// Note: VerifyChain scans pgtype.Timestamptz from rows.Scan.
// The stub above handles string/int types; for a full integration test against
// a real DB, use db-to-code.sh + testcontainers. The tests below cover the
// pure Go chain-verification logic via ComputeHash directly.

func TestVerifyChain_EmptyLedger(t *testing.T) {
	db := &stubDB{
		queryFunc: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &stubRows{}, nil
		},
	}
	res, err := presence.VerifyChain(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("empty ledger should be OK, got: %s", res.Error)
	}
}

// ── Tests unitaires de ComputeHash (partagés avec common/pkg/ledger) ─────────
// Ces tests valident la logique de hachage indépendamment de la DB.

func TestComputeHash_ChainLinkage(t *testing.T) {
	t0 := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)

	e1 := ledger.LedgerEntry{
		SeanceID:   42, UserID: 7, Statut: "PRESENT",
		EventAt:    t0,
		RecordedAt: t0.Add(50 * time.Millisecond),
		PrevHash:   ledger.GenesisHash,
	}
	h1 := ledger.ComputeHash(e1)

	e2 := ledger.LedgerEntry{
		SeanceID:   42, UserID: 8, Statut: "RETARD",
		EventAt:    t0.Add(5 * time.Minute),
		RecordedAt: t0.Add(5*time.Minute + 50*time.Millisecond),
		PrevHash:   h1,
	}
	h2 := ledger.ComputeHash(e2)

	if h1 == h2 {
		t.Fatal("distinct entries must have distinct hashes")
	}
	if h1 == ledger.GenesisHash {
		t.Fatal("hash must differ from genesis")
	}

	// Corruption détection : modifier e2.Statut change h2.
	e2corrupted := e2
	e2corrupted.Statut = "PRESENT"
	h2corrupted := ledger.ComputeHash(e2corrupted)
	if h2 == h2corrupted {
		t.Fatal("corrupting statut must change hash")
	}
}

func TestComputeHash_PrevHashCascade(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	e := ledger.LedgerEntry{
		SeanceID: 1, UserID: 1, Statut: "PRESENT",
		EventAt:    t0,
		RecordedAt: t0,
		PrevHash:   ledger.GenesisHash,
	}
	h := ledger.ComputeHash(e)

	// Simuler une altération du maillon précédent (prev_hash différent).
	eTampered := e
	eTampered.PrevHash = fmt.Sprintf("%064x", 1) // toute valeur != GenesisHash
	hTampered := ledger.ComputeHash(eTampered)

	if h == hTampered {
		t.Fatal("changing prev_hash must change the current hash")
	}
}

// ── Tests de configuration ────────────────────────────────────────────────────

func TestDefaultTSAURL(t *testing.T) {
	if services.DefaultTSAURL == "" {
		t.Fatal("DefaultTSAURL must not be empty")
	}
}
