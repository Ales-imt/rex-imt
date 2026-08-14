package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunSync exécute les six étapes dans l'ordre défini par le spec.
// Les étapes dépendant de l'année (promotions, planning, eleve_groupe) sont
// ignorées si aucune année de public.annee ne couvre la date du jour.
func RunSync(ctx context.Context, src source.Source, db *pgxpool.Pool) error {
	ac, ok, err := getAnneeCourante(ctx, New(db), time.Now())
	if err != nil {
		return err
	}
	if !ok {
		log.Printf("sync: aucune année ne correspond à la date du jour, étapes liées à l'année ignorées")
	}

	// 1. Promotions.
	if ok {
		if err := SyncPromotions(ctx, src, db, ac); err != nil {
			return err
		}
	}
	// 2. Profs (avant le planning pour que prof_id soit résolvable dans syncSeances).
	if err := SyncProfs(ctx, src, db); err != nil {
		return err
	}
	// 3 & 4. Planning (groupes + matières + séances + enrichissement des noms).
	if ok {
		if err := SyncPlanning(ctx, src, db, ac); err != nil {
			return err
		}
	}
	// 5. Élèves.
	if err := SyncEleves(ctx, src, db); err != nil {
		return err
	}
	// 6. Appartenance élève↔groupe.
	if ok {
		if err := SyncElevesGroupe(ctx, src, db, ac); err != nil {
			return err
		}
	}
	log.Printf("sync: cycle complet terminé")
	return nil
}
