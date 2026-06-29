package migration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const webdfdThrottle = 50 * time.Millisecond

// webdfdGet effectue un GET sur webdfd puis marque une pause pour ne pas saturer le serveur.
func webdfdGet(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	time.Sleep(webdfdThrottle)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

const (
	syncRefreshTTL = 2 * time.Hour
	syncRetryDelay = 5 * time.Minute
)

// Config regroupe les URLs nécessaires à la synchronisation webdfd.
type Config struct {
	PromosURL      string // ?TYPE=promos_txt
	CoursURL       string // ?TYPE=cours_txt
	PlanningURL    string // base URL pour planning_txt (sans paramètres)
	ElevesURL      string // ?TYPE=eleves_txt
	ProfsURL       string // ?TYPE=profs_txt
	ListeGroupeURL string // base URL pour listegroupe_html (cgihtml.exe)
}

// StartSync lance la goroutine de synchronisation complète au démarrage.
// En cas d'erreur, réessaie après syncRetryDelay ; en cas de succès,
// rafraîchit après syncRefreshTTL.
func StartSync(ctx context.Context, cfg Config, db *pgxpool.Pool) {
	go func() {
		for {
			if err := runSync(ctx, cfg, db); err != nil {
				log.Printf("sync: échec complet: %v — nouvel essai dans %s", err, syncRetryDelay)
				select {
				case <-time.After(syncRetryDelay):
				case <-ctx.Done():
					return
				}
				continue
			}
			select {
			case <-time.After(syncRefreshTTL):
			case <-ctx.Done():
				return
			}
		}
	}()
}

// runSync exécute les six étapes dans l'ordre défini par le spec.
func runSync(ctx context.Context, cfg Config, db *pgxpool.Pool) error {
	// 1. Promotions.
	if err := SyncPromotions(ctx, cfg.PromosURL, db); err != nil {
		return err
	}
	// 2. Profs (avant le planning pour que prof_id soit résolvable dans syncSeances).
	if err := SyncProfs(ctx, cfg.ProfsURL, db); err != nil {
		return err
	}
	// 3 & 4. Planning (groupes + matières + séances + enrichissement via cours_txt).
	if err := SyncPlanning(ctx, cfg.PlanningURL, db); err != nil {
		return err
	}
	// 5. Élèves.
	if err := SyncEleves(ctx, cfg.ElevesURL, db); err != nil {
		return err
	}
	// 6. Appartenance élève↔groupe.
	if err := SyncElevesGroupe(ctx, cfg.ListeGroupeURL, db); err != nil {
		return err
	}
	log.Printf("sync: cycle complet terminé")
	return nil
}
