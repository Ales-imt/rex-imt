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

// SyncEleves charge la liste globale des élèves depuis la source amont et
// maintient la correspondance dans migration.user_map.
//
// Ordre de résolution pour chaque EV :
//  1. Chercher dans user_map (external_id=EV) → mettre à jour les noms.
//  2. Sinon, chercher par email → créer uniquement la ligne user_map.
//  3. Sinon, créer l'utilisateur et la ligne user_map.
func SyncEleves(ctx context.Context, src source.Source, db *pgxpool.Pool) error {
	eleves, err := src.Eleves()
	if err != nil {
		return err
	}

	q := New(db)
	var count, created, linked int
	for _, e := range eleves {
		if e.Email == "" {
			log.Printf("eleve: EV=%s sans email, ignoré", e.ExternalID)
			continue
		}

		// Étape 1 : lookup par EV dans user_map.
		userID, err := q.GetUserBySource(ctx, GetUserBySourceParams{Source: source.Map, ExternalID: e.ExternalID})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user_map: lookup EV=%s: %w", e.ExternalID, err)
		}
		if err == nil {
			if err = q.UpdateUserNames(ctx, UpdateUserNamesParams{Name: e.Nom, Surname: e.Prenom, ID: userID}); err != nil {
				return err
			}
			if err = syncUserMap(ctx, q, userID, source.Map, e.ExternalID); err != nil {
				return err
			}
			count++
			continue
		}

		// Étape 2 : lookup par email.
		userID, err = q.GetUserByEmail(ctx, e.Email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user: lookup email=%s: %w", e.Email, err)
		}
		if err == nil {
			if err = syncUserMap(ctx, q, userID, source.Map, e.ExternalID); err != nil {
				return err
			}
			linked++
			count++
			continue
		}

		// Étape 3 : création de l'utilisateur.
		userID, err = createUserEleve(ctx, db, e.Nom, e.Prenom, e.Email)
		if err != nil {
			return err
		}
		if err = syncUserMap(ctx, q, userID, source.Map, e.ExternalID); err != nil {
			return err
		}
		created++
		count++
	}
	log.Printf("eleve: %d élèves synchronisés (%d créés, %d rattachés par email)", count, created, linked)
	return nil
}
