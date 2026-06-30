package pointage

import (
	"back-rex-common/pkg/ledger"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ledgerAdvisoryKey is the PostgreSQL advisory lock key used to serialise all
// ledger writes. Any value is valid as long as it is stable and unique within
// the application. Using a transaction-level lock (pg_advisory_xact_lock) means
// it is released automatically on commit or rollback.
const ledgerAdvisoryKey = 9_001_001

// AppendLedger adds a new entry to presence_ledger within tx.
// tx MUST already be open; commit/rollback is the caller's responsibility.
//
// Sérialisation : pg_advisory_xact_lock serialises concurrent writes so that
// no two entries ever share the same prev_hash (which would break the chain).
//
// recorded_at is set to time.Now().UTC() inside this function (never from the
// client) and is passed explicitly to the INSERT so its value is known before
// hash computation.
func AppendLedger(ctx context.Context, tx pgx.Tx, seanceID int64, userID int32, statut string, eventAt time.Time) (seq int64, h string, err error) {
	// 1. Serialise concurrent writers.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", ledgerAdvisoryKey); err != nil {
		return 0, "", fmt.Errorf("ledger advisory lock: %w", err)
	}

	// 2. Fetch the previous hash (genesis sentinel when the ledger is empty).
	prevHash := ledger.GenesisHash
	scanErr := tx.QueryRow(ctx,
		"SELECT hash FROM presence_ledger ORDER BY seq DESC LIMIT 1",
	).Scan(&prevHash)
	if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		return 0, "", fmt.Errorf("ledger get last hash: %w", scanErr)
	}

	// 3. Compute hash.
	recordedAt := time.Now().UTC()
	h = ledger.ComputeHash(ledger.LedgerEntry{
		SeanceID:   seanceID,
		UserID:     userID,
		Statut:     statut,
		EventAt:    eventAt.UTC(),
		RecordedAt: recordedAt,
		PrevHash:   prevHash,
	})

	// 4. Insert and retrieve the assigned seq.
	err = tx.QueryRow(ctx,
		`INSERT INTO presence_ledger
		 (seance_id, user_id, statut, event_at, recorded_at, prev_hash, hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING seq`,
		seanceID, userID, statut, eventAt.UTC(), recordedAt, prevHash, h,
	).Scan(&seq)
	if err != nil {
		return 0, "", fmt.Errorf("ledger insert: %w", err)
	}
	return seq, h, nil
}
