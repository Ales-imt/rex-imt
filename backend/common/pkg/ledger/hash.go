package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GenesisHash is the conventional prev_hash of the first ledger entry.
// 64 zero digits make it visually recognisable as the chain origin.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// LedgerEntry carries the fields that enter the hash computation.
// The field order below is contractual: changing it invalidates the entire chain.
type LedgerEntry struct {
	SeanceID   int64
	UserID     int32
	Statut     string
	EventAt    time.Time // business timestamp (pointe_at), must be UTC
	RecordedAt time.Time // server-side recording timestamp, must be UTC
	PrevHash   string    // GenesisHash for the very first entry
}

// ComputeHash returns the hex-encoded SHA-256 of the canonical serialisation of e.
//
// Canonical format (pipe-separated, field order is contractual):
//
//	seance_id|user_id|statut|event_at|recorded_at|prev_hash
//
// Timestamps are formatted as RFC3339Nano in UTC (nanosecond precision,
// e.g. "2026-06-29T14:30:00.123456789Z"). Using a fixed format avoids any
// platform-dependent time.String() ambiguity.
func ComputeHash(e LedgerEntry) string {
	canonical := fmt.Sprintf("%d|%d|%s|%s|%s|%s",
		e.SeanceID,
		e.UserID,
		e.Statut,
		e.EventAt.UTC().Format(time.RFC3339Nano),
		e.RecordedAt.UTC().Format(time.RFC3339Nano),
		e.PrevHash,
	)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}
