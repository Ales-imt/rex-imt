package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SyncElevesGroupe synchronise l'appartenance élève↔groupe pour tous les groupes
// amont connus de l'année courante.
//
// Pour chaque groupe :
//  1. Récupère la liste des membres auprès de la source.
//  2. Valide le libellé du groupe quand la source le fournit.
//  3. Résout chaque EV via migration.user_map ; ignore les EV inconnus (log + skip).
//  4. Upsert eleve_groupe.
//  5. Supprime les entrées eleve_groupe qui ne sont plus dans la liste.
//  6. Met à jour groupe.taille.
func SyncElevesGroupe(ctx context.Context, src source.Source, db *pgxpool.Pool, ac anneeCourante) error {
	q := New(db)
	annee := ac.Annee

	groupes, err := q.ListGroupeWebdfdIDs(ctx, annee)
	if err != nil {
		return err
	}

	for _, gr := range groupes {
		if err := syncGroupe(ctx, src, gr.InternalID, gr.ExternalID, q); err != nil {
			log.Printf("eleve_groupe: groupe GRCLE=%s: %v — continué", gr.ExternalID, err)
		}
	}
	return nil
}

func syncGroupe(ctx context.Context, src source.Source, internalID int64, grcle string, q *Queries) error {
	evs, groupeLabel, err := src.MembresGroupe(grcle)
	if err != nil {
		return err
	}

	// Garde-fou : vérifier la cohérence du libellé quand la source le porte.
	if groupeLabel != "" {
		expectedLabel, err := q.GetGroupeLabel(ctx, internalID)
		if err == nil && expectedLabel != "" && !strings.EqualFold(groupeLabel, expectedLabel) {
			log.Printf("eleve_groupe: GRCLE=%s: libellé flux %q ≠ DB %q (GRCLE périmé ?)", grcle, groupeLabel, expectedLabel)
		}
	}

	idGroupe := int32(internalID)
	var seen []int32
	var skipped int
	for _, ev := range evs {
		userID, err := q.GetUserBySource(ctx, GetUserBySourceParams{Source: source.Map, ExternalID: ev})
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("eleve_groupe: EV=%s introuvable dans user_map, ignoré (rechargez les élèves)", ev)
			skipped++
			continue
		}
		if err != nil {
			return fmt.Errorf("user_map: lookup EV=%s: %w", ev, err)
		}
		if err = q.UpsertEleveGroupe(ctx, UpsertEleveGroupeParams{NumEtudiant: userID, IDGroupe: idGroupe}); err != nil {
			return err
		}
		seen = append(seen, userID)
	}

	// Réconciliation : retirer les membres absents de la liste courante.
	if err = q.DeleteEleveGroupeAbsents(ctx, DeleteEleveGroupeAbsentsParams{
		IDGroupe: idGroupe,
		Column2:  seen,
	}); err != nil {
		return err
	}

	taille := len(seen)
	if err = q.UpdateGroupeTaille(ctx, UpdateGroupeTailleParams{Taille: int32(taille), ID: internalID}); err != nil {
		return err
	}

	log.Printf("eleve_groupe: GRCLE=%s: %d élèves (%d ignorés)", grcle, taille, skipped)
	return nil
}
