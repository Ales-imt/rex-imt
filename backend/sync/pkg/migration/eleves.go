package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// syncEleves maintient la correspondance des élèves dans migration.user_map.
//
// Ordre de résolution pour chaque EV :
//  1. Chercher dans user_map (external_id=EV) → mettre à jour les noms.
//  2. Sinon, et seulement si l'élève appartient à un groupe du planning,
//     chercher par email → compléter le compte + créer la ligne user_map.
//  3. Sinon, et sous la même condition, créer l'utilisateur et la ligne.
//
// Comme pour les profs, seules les étapes 2 et 3 sont filtrées : un élève déjà
// connu reste rafraîchi.
func syncEleves(ctx context.Context, q *Queries, c *collecte) error {
	f := c.filtreEleves()

	var count, created, linked, ecartes int
	for _, e := range c.eleves {
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

		// Inconnu ET absent de tout groupe du planning : on n'ouvre rien.
		if !f.retient(e.ExternalID) {
			ecartes++
			continue
		}

		// Étape 2 : lookup par email.
		//
		// Le compte existe déjà — souvent parce que la personne est aussi
		// professeur, ou qu'elle a été créée par un autre chemin. Il faut le
		// compléter, pas seulement le référencer : sans rôle ELEVE ni ligne
		// public.student, ce compte n'est pas un élève pour le reste de
		// l'application, alors que user_map affirme le contraire.
		userID, err = q.GetUserByEmail(ctx, e.Email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user: lookup email=%s: %w", e.Email, err)
		}
		if err == nil {
			if err = q.AddEleveRole(ctx, userID); err != nil {
				return err
			}
			if err = q.InsertStudent(ctx, userID); err != nil {
				return fmt.Errorf("student: rattachement user_id=%d: %w", userID, err)
			}
			if err = syncUserMap(ctx, q, userID, source.Map, e.ExternalID); err != nil {
				return err
			}
			linked++
			count++
			continue
		}

		// Étape 3 : création de l'utilisateur.
		userID, err = createUserEleve(ctx, q, e.Nom, e.Prenom, e.Email)
		if err != nil {
			return err
		}
		if err = syncUserMap(ctx, q, userID, source.Map, e.ExternalID); err != nil {
			return err
		}
		created++
		count++
	}
	log.Printf("eleve: %d élèves synchronisés (%d créés, %d rattachés par email, %d écartés faute de groupe)",
		count, created, linked, ecartes)
	return nil
}
