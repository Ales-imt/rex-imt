package ledger

// Accès au registre presence_ledger — UNIQUE implémentation du chaînage,
// partagée entre back-rex-admin (ancrage TSA, vérification) et back-rex-eleve
// (pointage). Le hash canonique est dans hash.go ; les requêtes SQL sont
// générées par sqlc dans pkg/ledger/gen.

import (
	"back-rex-common/pkg/ledger/gen"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// DBTX est l'abstraction d'accès DB des fonctions de lecture du registre,
// satisfaite par *pgxpool.Pool comme par pgx.Tx.
type DBTX = gen.DBTX

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

	q := gen.New(tx)

	// 2. Fetch the previous hash (genesis sentinel when the ledger is empty).
	prevHash := GenesisHash
	last, scanErr := q.GetLastLedgerEntry(ctx)
	if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
		return 0, "", fmt.Errorf("ledger get last hash: %w", scanErr)
	}
	if scanErr == nil {
		prevHash = last.Hash
	}

	// 3. Compute hash. Les timestamps sont tronqués à la microseconde AVANT
	// l'insertion : la valeur envoyée est déjà exactement représentable en
	// timestamptz, sans dépendre du driver (pgx tronque en binaire, mais le
	// protocole texte arrondirait — ce qui casserait la re-vérification).
	eventAt = eventAt.UTC().Truncate(time.Microsecond)
	recordedAt := time.Now().UTC().Truncate(time.Microsecond)
	h = ComputeHash(LedgerEntry{
		SeanceID:   seanceID,
		UserID:     userID,
		Statut:     statut,
		EventAt:    eventAt,
		RecordedAt: recordedAt,
		PrevHash:   prevHash,
	})

	// 4. Insert and retrieve the assigned seq.
	seq, err = q.InsertLedgerEntry(ctx, gen.InsertLedgerEntryParams{
		SeanceID:   seanceID,
		UserID:     userID,
		Statut:     statut,
		EventAt:    pgtype.Timestamptz{Time: eventAt.UTC(), Valid: true},
		RecordedAt: pgtype.Timestamptz{Time: recordedAt, Valid: true},
		PrevHash:   prevHash,
		Hash:       h,
	})
	if err != nil {
		return 0, "", fmt.Errorf("ledger insert: %w", err)
	}
	return seq, h, nil
}

// GetLastLedgerEntry returns the seq and hash of the last ledger entry.
// Returns pgx.ErrNoRows when the ledger is empty.
func GetLastLedgerEntry(ctx context.Context, db DBTX) (seq int64, hash string, err error) {
	last, err := gen.New(db).GetLastLedgerEntry(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, GenesisHash, pgx.ErrNoRows
	}
	return last.Seq, last.Hash, err
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
	rows, err := gen.New(db).ListLedgerBySeq(ctx)
	if err != nil {
		return VerifyChainResult{}, fmt.Errorf("ledger query: %w", err)
	}

	prevHash := GenesisHash // hash of the previous row, genesis for the first

	for _, row := range rows {
		// Check prev_hash linkage.
		if row.PrevHash != prevHash {
			return VerifyChainResult{
				OK:       false,
				BrokenAt: row.Seq,
				Error:    fmt.Sprintf("seq %d: prev_hash mismatch (got %s, want %s)", row.Seq, row.PrevHash, prevHash),
			}, nil
		}

		// Recompute and check hash.
		computed := ComputeHash(LedgerEntry{
			SeanceID:   row.SeanceID,
			UserID:     row.UserID,
			Statut:     row.Statut,
			EventAt:    row.EventAt.Time.UTC(),
			RecordedAt: row.RecordedAt.Time.UTC(),
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
	}
	return VerifyChainResult{OK: true}, nil
}
