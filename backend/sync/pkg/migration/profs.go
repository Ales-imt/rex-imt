package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SyncProfs charge la liste des professeurs depuis la source amont et
// maintient la correspondance dans migration.prof_map.
//
// Ordre de résolution pour chaque PRCLE :
//  1. Chercher dans prof_map (external_id=PRCLE) → mettre à jour les noms.
//  2. Sinon, chercher par email → ajouter le rôle PROF + créer la ligne prof_map.
//  3. Sinon, créer l'utilisateur avec rôle PROF et la ligne prof_map.
func SyncProfs(ctx context.Context, src source.Source, db *pgxpool.Pool) error {
	profs, err := src.Profs()
	if err != nil {
		return err
	}

	q := New(db)
	var count, created, linked int
	for _, p := range profs {
		if p.Email == "" {
			log.Printf("prof: PRCLE=%s sans email, ignoré", p.ExternalID)
			continue
		}

		// Étape 1 : lookup par PRCLE dans prof_map.
		userID, err := q.GetProfBySource(ctx, GetProfBySourceParams{Source: source.Map, ExternalID: p.ExternalID})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("prof_map: lookup PRCLE=%s: %w", p.ExternalID, err)
		}
		if err == nil {
			if err = q.UpdateUserNames(ctx, UpdateUserNamesParams{Name: p.Nom, Surname: p.Prenom, ID: userID}); err != nil {
				return err
			}
			if err = syncProfMap(ctx, q, userID, source.Map, p.ExternalID); err != nil {
				return err
			}
			count++
			continue
		}

		// Étape 2 : lookup par email.
		userID, err = q.GetUserByEmail(ctx, p.Email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user: lookup email=%s: %w", p.Email, err)
		}
		if err == nil {
			if err = q.AddProfRole(ctx, userID); err != nil {
				return err
			}
			if err = syncProfMap(ctx, q, userID, source.Map, p.ExternalID); err != nil {
				return err
			}
			linked++
			count++
			continue
		}

		// Étape 3 : création de l'utilisateur.
		userID, err = q.CreateUserProf(ctx, CreateUserProfParams{Email: p.Email, Name: p.Nom, Surname: p.Prenom})
		if err != nil {
			return err
		}
		if err = syncProfMap(ctx, q, userID, source.Map, p.ExternalID); err != nil {
			return err
		}
		created++
		count++
	}
	log.Printf("prof: %d profs synchronisés (%d créés, %d rattachés par email)", count, created, linked)
	return nil
}
