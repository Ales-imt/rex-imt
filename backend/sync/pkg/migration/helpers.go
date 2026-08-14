package migration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// anneeCourante décrit l'année scolaire (table public.annee) couvrant une date donnée.
type anneeCourante struct {
	ID    int64
	Annee int32 // année civile de début, utilisée comme clé naturelle des tables migration.*_map
	Debut time.Time
	Fin   time.Time
}

// getAnneeCourante cherche dans public.annee l'année dont la période [debut, fin]
// contient `now`. ok vaut false si aucune année ne correspond.
func getAnneeCourante(ctx context.Context, q *Queries, now time.Time) (ac anneeCourante, ok bool, err error) {
	row, err := q.GetAnneeByDate(ctx, now)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return anneeCourante{}, false, nil
		}
		return anneeCourante{}, false, fmt.Errorf("annee: lookup: %w", err)
	}
	return anneeCourante{ID: row.ID, Annee: int32(row.Debut.Year()), Debut: row.Debut, Fin: row.Fin}, true, nil
}

func createUserEleve(ctx context.Context, db *pgxpool.Pool, name, surname, email string) (int32, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("user: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	q := New(tx)
	id, err := q.InsertUserEleve(ctx, InsertUserEleveParams{Email: email, Name: name, Surname: surname})
	if err != nil {
		return 0, fmt.Errorf("user: create %s: %w", email, err)
	}
	if err = q.InsertStudent(ctx, id); err != nil {
		return 0, fmt.Errorf("student: create user_id=%d: %w", id, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("user: commit: %w", err)
	}
	return id, nil
}

func syncProfMap(ctx context.Context, q *Queries, internalID int32, source, extID string) error {
	if err := q.DeleteStaleProfMap(ctx, DeleteStaleProfMapParams{
		InternalID: internalID,
		Source:     source,
		ExternalID: extID,
	}); err != nil {
		return fmt.Errorf("prof_map: delete stale source=%s internal=%d: %w", source, internalID, err)
	}
	if err := q.UpsertProfMap(ctx, UpsertProfMapParams{
		InternalID: internalID,
		Source:     source,
		ExternalID: extID,
	}); err != nil {
		return fmt.Errorf("prof_map: insert source=%s ext=%s: %w", source, extID, err)
	}
	return nil
}

func syncUserMap(ctx context.Context, q *Queries, internalID int32, source, extID string) error {
	if err := q.DeleteStaleUserMap(ctx, DeleteStaleUserMapParams{
		InternalID: internalID,
		Source:     source,
		ExternalID: extID,
	}); err != nil {
		return fmt.Errorf("user_map: delete stale source=%s internal=%d: %w", source, internalID, err)
	}
	if err := q.UpsertUserMap(ctx, UpsertUserMapParams{
		InternalID: internalID,
		Source:     source,
		ExternalID: extID,
	}); err != nil {
		return fmt.Errorf("user_map: insert source=%s ext=%s: %w", source, extID, err)
	}
	return nil
}
