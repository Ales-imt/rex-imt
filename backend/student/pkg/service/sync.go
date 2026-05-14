package service

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartSync charge les promotions puis les matières depuis webdfd au démarrage,
// réessaie après 5 minutes en cas d'erreur, et rafraîchit toutes les 2 heures.
func StartSync(ctx context.Context, promotionURL, matiereURL string, db *pgxpool.Pool) {
	go func() {
		for {
			if err := syncPromotions(ctx, promotionURL, db); err != nil {
				log.Printf("sync: promotions échouées: %v — nouvel essai dans %s", err, matiereRetryDelay)
				select {
				case <-time.After(matiereRetryDelay):
				case <-ctx.Done():
					return
				}
				continue
			}

			if err := syncMatieres(ctx, matiereURL, db); err != nil {
				log.Printf("sync: matières échouées: %v — nouvel essai dans %s", err, matiereRetryDelay)
				select {
				case <-time.After(matiereRetryDelay):
				case <-ctx.Done():
					return
				}
				continue
			}

			select {
			case <-time.After(matiereRefreshTTL):
			case <-ctx.Done():
				return
			}
		}
	}()
}
