package auth

// Tests de la fenêtre de grâce du refresh (jetonEnGrace) : un jeton déjà
// consommé reste échangeable tant que sa consommation date de moins de
// refreshGraceWindow — le cas de la réponse de rotation perdue. Fake DBTX en
// mémoire, sur le même principe que fakeAuthDB (email_otp_test.go).

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// fakeGraceDB ne sert que GetRefreshTokenByPrev : une seule ligne vivante,
// indexée par le hachage de son prédécesseur.
type fakeGraceDB struct {
	vivant *RefreshToken
}

func (f *fakeGraceDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected Exec: " + sql)
}

func (f *fakeGraceDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query: " + sql)
}

func (f *fakeGraceDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if !strings.Contains(sql, "WHERE prev_token") {
		return &fakeRow{err: errors.New("unexpected QueryRow: " + sql)}
	}
	prev := args[0].(pgtype.Text)
	if f.vivant == nil || !f.vivant.PrevToken.Valid || f.vivant.PrevToken.String != prev.String {
		return &fakeRow{err: pgx.ErrNoRows}
	}
	return &fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*int32)) = f.vivant.ID
		*(dest[1].(*int32)) = f.vivant.UserID
		*(dest[2].(*string)) = f.vivant.Token
		*(dest[3].(*pgtype.Timestamptz)) = f.vivant.ExpiresAt
		*(dest[4].(*pgtype.Timestamptz)) = f.vivant.CreatedAt
		*(dest[5].(*bool)) = f.vivant.Revoked
		*(dest[6].(*pgtype.Int4)) = f.vivant.TokenVersion
		*(dest[7].(*string)) = f.vivant.Session
		*(dest[8].(*pgtype.Text)) = f.vivant.PrevToken
		*(dest[9].(*pgtype.Timestamptz)) = f.vivant.PrevConsumedAt
		return nil
	}}
}

func ligneVivante(consommeIlYA time.Duration) *RefreshToken {
	instant := time.Now().Add(-consommeIlYA)
	return &RefreshToken{
		ID:             7,
		UserID:         4185,
		Token:          "hash-v5",
		ExpiresAt:      pgtype.Timestamptz{Time: time.Now().Add(90 * 24 * time.Hour), Valid: true},
		Revoked:        false,
		TokenVersion:   pgtype.Int4{Int32: 5, Valid: true},
		Session:        "session-test",
		PrevToken:      pgtype.Text{String: "hash-v4", Valid: true},
		PrevConsumedAt: pgtype.Timestamptz{Time: instant, Valid: true},
	}
}

func TestJetonEnGrace_DansLaFenetre(t *testing.T) {
	db := &fakeGraceDB{vivant: ligneVivante(10 * time.Second)}
	row, err := jetonEnGrace(context.Background(), New(db), "hash-v4")
	if err != nil {
		t.Fatalf("grâce refusée à 10 s alors que la fenêtre est de %v : %v", refreshGraceWindow, err)
	}
	if row.TokenVersion.Int32 != 5 || row.Session != "session-test" {
		t.Fatalf("la grâce doit rendre la ligne vivante (v5), reçu v%d session=%q", row.TokenVersion.Int32, row.Session)
	}
}

func TestJetonEnGrace_FenetreDepassee(t *testing.T) {
	db := &fakeGraceDB{vivant: ligneVivante(refreshGraceWindow + time.Second)}
	if _, err := jetonEnGrace(context.Background(), New(db), "hash-v4"); err != pgx.ErrNoRows {
		t.Fatalf("hors fenêtre, attendu ErrNoRows (le refus reprend ses droits), reçu %v", err)
	}
}

func TestJetonEnGrace_PredecesseurInconnu(t *testing.T) {
	db := &fakeGraceDB{vivant: ligneVivante(10 * time.Second)}
	if _, err := jetonEnGrace(context.Background(), New(db), "hash-v3"); err != pgx.ErrNoRows {
		t.Fatalf("un jeton qui n'est pas le prédécesseur direct ne doit pas passer, reçu %v", err)
	}
}

func TestJetonEnGrace_SansAncrage(t *testing.T) {
	vivant := ligneVivante(10 * time.Second)
	vivant.PrevConsumedAt = pgtype.Timestamptz{} // ligne d'avant la migration
	db := &fakeGraceDB{vivant: vivant}
	if _, err := jetonEnGrace(context.Background(), New(db), "hash-v4"); err != pgx.ErrNoRows {
		t.Fatalf("sans prev_consumed_at, pas de grâce possible, reçu %v", err)
	}
}
