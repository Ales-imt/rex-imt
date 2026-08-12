// Package presencedata porte la donnée de présence partagée entre
// back-rex-admin (page web Pointage) et back-rex-eleve (écrans mobiles
// prof/gestionnaire) : les requêtes sqlc du sous-paquet gen, et les rares
// opérations qui composent plusieurs de ces requêtes en une seule transaction.
package presencedata

import (
	"back-rex-common/pkg/presencedata/gen"
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CloseSeanceEtFiger clôture une séance et fige son effectif attendu.
//
// Les deux écritures vivent ici, et non dans les handlers, parce que DEUX
// chemins clôturent une séance — CloseSeanceHandler côté admin et
// CloseSeanceProfHandler côté mobile. Dupliquer le figement dans les deux
// laisserait tôt ou tard des séances clôturées sans effectif figé, selon le
// bouton utilisé.
//
// L'effectif n'est figé que si la clôture a RÉELLEMENT eu lieu : la clôture est
// idempotente (WHERE closed_at IS NULL), un second appel ne doit donc ni
// dupliquer de lignes ni réécrire figee_at avec l'effectif du jour.
//
// Les deux opérations sont dans une seule transaction : une séance clôturée
// dont le figement aurait échoué garderait un effectif vivant, c'est-à-dire
// une feuille officielle qui continue de bouger.
//
// Retourne ferme=false si la séance était déjà clôturée.
func CloseSeanceEtFiger(ctx context.Context, db *pgxpool.Pool, seanceID int64) (ferme bool, figes int64, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("clôture séance %d: begin: %w", seanceID, err)
	}
	defer tx.Rollback(ctx)

	q := gen.New(tx)

	n, err := q.CloseSeance(ctx, seanceID)
	if err != nil {
		return false, 0, fmt.Errorf("clôture séance %d: %w", seanceID, err)
	}
	if n == 0 {
		// Déjà clôturée : son effectif est figé depuis le premier appel.
		return false, 0, nil
	}

	figes, err = q.FigerEffectifSeance(ctx, seanceID)
	if err != nil {
		return false, 0, fmt.Errorf("figement effectif séance %d: %w", seanceID, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return false, 0, fmt.Errorf("clôture séance %d: commit: %w", seanceID, err)
	}

	if figes == 0 {
		// Aucun attendu calculable (matière sans période, groupe pas encore
		// synchronisé…). On n'insère rien VOLONTAIREMENT : une séance figée à
		// zéro serait indistinguable d'une séance sans attendus et gèlerait une
		// feuille vide pour toujours. Sans ligne, la vue retombe sur sa branche
		// vivante et la feuille se remplira dès que la donnée arrivera.
		log.Printf("présence: séance %d clôturée avec un effectif attendu vide — effectif non figé", seanceID)
	}
	return true, figes, nil
}
