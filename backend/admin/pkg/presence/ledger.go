package presence

import (
	"back-rex-common/pkg/ledger"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ledgerRow is the in-memory representation of a presence_ledger row used by VerifyChain.
type ledgerRow struct {
	Seq        int64
	SeanceID   int64
	UserID     int32
	Statut     string
	EventAt    time.Time
	RecordedAt time.Time
	PrevHash   string
	Hash       string
}

// VerifyChainResult is the result of a chain verification.
type VerifyChainResult struct {
	OK       bool
	BrokenAt int64  // seq of the first broken entry; 0 when OK
	Error    string // human-readable description when not OK
}

// VerifyChain reads every entry in presence_ledger ordered by seq and
// recomputes each hash, verifying that:
//  1. ComputeHash(entry) == entry.Hash
//  2. entry.PrevHash == hash of the previous entry (or GenesisHash for the first)
//
// Returns OK=true when the chain is intact. On the first broken link it returns
// OK=false and BrokenAt set to the seq of the faulty entry.
func VerifyChain(ctx context.Context, db DBTX) (VerifyChainResult, error) {
	rows, err := db.Query(ctx,
		`SELECT seq, seance_id, user_id, statut, event_at, recorded_at, prev_hash, hash
		 FROM presence_ledger ORDER BY seq ASC`,
	)
	if err != nil {
		return VerifyChainResult{}, fmt.Errorf("ledger query: %w", err)
	}
	defer rows.Close()

	var prevHash string // will be set to the hash of the previous row
	first := true

	for rows.Next() {
		var row ledgerRow
		var eventAt, recordedAt pgtype.Timestamptz
		if err := rows.Scan(
			&row.Seq, &row.SeanceID, &row.UserID, &row.Statut,
			&eventAt, &recordedAt, &row.PrevHash, &row.Hash,
		); err != nil {
			return VerifyChainResult{}, fmt.Errorf("ledger scan seq=%d: %w", row.Seq, err)
		}
		row.EventAt = eventAt.Time.UTC()
		row.RecordedAt = recordedAt.Time.UTC()

		// Check prev_hash linkage.
		expectedPrev := ledger.GenesisHash
		if !first {
			expectedPrev = prevHash
		}
		if row.PrevHash != expectedPrev {
			return VerifyChainResult{
				OK:       false,
				BrokenAt: row.Seq,
				Error:    fmt.Sprintf("seq %d: prev_hash mismatch (got %s, want %s)", row.Seq, row.PrevHash, expectedPrev),
			}, nil
		}

		// Recompute and check hash.
		computed := ledger.ComputeHash(ledger.LedgerEntry{
			SeanceID:   row.SeanceID,
			UserID:     row.UserID,
			Statut:     row.Statut,
			EventAt:    row.EventAt,
			RecordedAt: row.RecordedAt,
			PrevHash:   row.PrevHash,
		})
		if computed != row.Hash {
			return VerifyChainResult{
				OK:       false,
				BrokenAt: row.Seq,
				Error:    fmt.Sprintf("seq %d: hash mismatch (stored %s, computed %s)", row.Seq, row.Hash, computed),
			}, nil
		}

		prevHash = row.Hash
		first = false
	}
	if err := rows.Err(); err != nil {
		return VerifyChainResult{}, fmt.Errorf("ledger rows: %w", err)
	}
	return VerifyChainResult{OK: true}, nil
}

// GetLastLedgerEntry returns the seq and hash of the last ledger entry.
// Returns pgx.ErrNoRows when the ledger is empty.
func GetLastLedgerEntry(ctx context.Context, db DBTX) (seq int64, hash string, err error) {
	err = db.QueryRow(ctx,
		"SELECT seq, hash FROM presence_ledger ORDER BY seq DESC LIMIT 1",
	).Scan(&seq, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ledger.GenesisHash, pgx.ErrNoRows
	}
	return seq, hash, err
}
