// Package scheduler décide QUAND synchroniser. Le moteur (pkg/migration) sait
// comment, les sources (pkg/source) savent où lire ; ici on ne traite que le
// déclenchement, qui diffère du tout au tout d'une source à l'autre.
package scheduler

import (
	"back-rex-common/pkg/hfsqlexport"
	"back-rex-common/pkg/services"
	"back-rex-sync/pkg/migration"
	"back-rex-sync/pkg/source/connect"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// RafraichissementWebdfdDefaut : cybema ne prévient pas de ses changements,
	// il faut donc l'interroger régulièrement. Surchargeable par
	// migration.webdfd.interval.
	RafraichissementWebdfdDefaut = 2 * time.Hour
	// ReessaiWebdfdDefaut : délai après un cycle en échec (coupure réseau,
	// cgiempt indisponible). Surchargeable par migration.webdfd.retry.
	ReessaiWebdfdDefaut = 5 * time.Minute
	// SondageHFSQLDefaut : à quelle fréquence chercher un nouvel export.
	SondageHFSQLDefaut = time.Minute
)

// Periodique boucle sur la source webdfd : un cycle, puis attente.
func Periodique(ctx context.Context, cfg *services.Config, db *pgxpool.Pool) error {
	src, err := connect.Nouvelle(cfg, "")
	if err != nil {
		return err
	}

	rafraichissement := cfg.Migration.Webdfd.Interval
	if rafraichissement <= 0 {
		rafraichissement = RafraichissementWebdfdDefaut
	}
	reessai := cfg.Migration.Webdfd.Retry
	if reessai <= 0 {
		reessai = ReessaiWebdfdDefaut
	}
	log.Printf("sync: source webdfd, un cycle toutes les %s (réessai à %s après échec)", rafraichissement, reessai)

	dump := cfg.Migration.DumpPlanning
	if dump != "" {
		log.Printf("sync: dump de diagnostic du planning activé → %s", dump)
	}

	go func() {
		for {
			attente := rafraichissement
			if err := migration.RunSync(ctx, src, db, dump); err != nil {
				log.Printf("sync: échec complet: %v — nouvel essai dans %s", err, reessai)
				attente = reessai
			}
			select {
			case <-time.After(attente):
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// SurNouvelExport boucle sur la source hfsql : un cycle à chaque export complet
// apparu sous data.
//
// Le sondage est préféré à inotify parce qu'il est déclenché sur NIVEAU et non
// sur front : il ne demande jamais « que s'est-il passé ? » mais « quel est le
// dernier export complet ? ». Un événement perdu — file du noyau saturée,
// service redémarré pendant que l'export se termine — laisserait au contraire
// un instantané complet que plus personne ne viendrait lire, jusqu'au dépôt
// suivant et sans que rien ne le signale.
//
// Le prix est une latence d'au plus un intervalle, sur un événement qui
// survient une fois par jour.
//
// L'erreur rendue est celle du démarrage — configuration invalide. Passé ce
// point, tout est journalisé et la boucle continue : un export illisible ne
// doit pas empêcher le suivant d'être pris.
func SurNouvelExport(ctx context.Context, cfg *services.Config, db *pgxpool.Pool) error {
	data := cfg.Migration.HFSQL.Data
	if data == "" {
		return errChampObligatoire("migration.hfsql.data")
	}
	intervalle := cfg.Migration.HFSQL.Interval
	if intervalle <= 0 {
		intervalle = SondageHFSQLDefaut
	}

	go func() {
		// Aucun état persisté : au redémarrage on resynchronise le dernier
		// export. Les étapes sont idempotentes (upsert), donc rejouer un
		// instantané déjà vu ne coûte qu'un cycle.
		var dernierVu string
		ticker := time.NewTicker(intervalle)
		defer ticker.Stop()

		for {
			nom, jsonDir, err := hfsqlexport.Dernier(data)
			switch {
			case err != nil:
				// Répertoire absent : le producteur n'a peut-être pas encore
				// démarré. On le redit à chaque tour, sans arrêter le service.
				log.Printf("sync: %v", err)
			case nom == "":
				// Rien de complet pour l'instant : silence, c'est l'état normal
				// avant le premier dépôt.
			case nom != dernierVu:
				log.Printf("sync: nouvel export %s", nom)
				if synchroniser(ctx, cfg, jsonDir, db) {
					dernierVu = nom
				}
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// synchroniser monte la source sur un export et lance un cycle complet.
// Rend false si le cycle a échoué, pour que l'export soit retenté au tour
// suivant plutôt que considéré comme traité.
func synchroniser(ctx context.Context, cfg *services.Config, jsonDir string, db *pgxpool.Pool) bool {
	src, err := connect.Nouvelle(cfg, jsonDir)
	if err != nil {
		log.Printf("sync: export %s illisible: %v", jsonDir, err)
		return false
	}
	if err := migration.RunSync(ctx, src, db, cfg.Migration.DumpPlanning); err != nil {
		log.Printf("sync: synchronisation depuis %s: %v", jsonDir, err)
		return false
	}
	return true
}

type errChampObligatoire string

func (e errChampObligatoire) Error() string { return string(e) + " obligatoire" }
