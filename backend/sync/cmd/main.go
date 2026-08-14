// Commande rex-sync : maintient dans PostgreSQL le référentiel scolaire
// (promotions, profs, élèves, planning, groupes) à partir de la source amont
// désignée par la configuration.
//
// Deux sources, deux déclenchements :
//   - webdfd : cybema interrogé en HTTP, rafraîchi toutes les 2 h ;
//   - hfsql  : les exports JSON déposés par le service export-webdfd, pris à
//     mesure qu'ils apparaissent.
//
// Le service n'expose qu'un point de santé : il ne sert aucune donnée, il écrit
// en base et c'est tout.
package main

import (
	"back-rex-common/pkg/health"
	"back-rex-common/pkg/services"
	"back-rex-sync/pkg/scheduler"
	"back-rex-sync/pkg/source"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	// Embarque la base IANA des fuseaux : l'image runtime ne fournit pas les
	// zones, et sans elle LoadLocation("Europe/Paris") retombe sur UTC — tout le
	// planning serait décalé.
	_ "time/tzdata"
)

// Injectées à la compilation.
var (
	buildTime string
	version   string
)

func main() {
	startTime := time.Now()
	log.Printf("rex-sync version: %s", version)
	log.Printf("Compilation time: %s", buildTime)

	configPath := "/opt/rex-sync/conf/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	cfg, err := services.LoadConfigYaml(configPath)
	if err != nil {
		log.Fatal("Erreur chargement config YAML :", err)
	}

	ctx := context.Background()
	pg := services.NewPG(ctx, services.ToDBS(&cfg.Database))
	services.ConnectWithRetry("PostgreSQL", 5*time.Minute, func() error {
		return pg.Ping(ctx)
	})

	// Une source absente ou inconnue arrête le démarrage : se rabattre sur
	// l'autre synchroniserait un référentiel différent de celui attendu, sans
	// que personne ne s'en aperçoive.
	switch cfg.Migration.Source {
	case source.NomWebdfd:
		err = scheduler.Periodique(ctx, cfg, pg.Db)
	case source.NomHFSQL:
		err = scheduler.SurNouvelExport(ctx, cfg, pg.Db)
	case "":
		log.Fatalf("migration: aucune source configurée (migration.source : %s ou %s)",
			source.NomWebdfd, source.NomHFSQL)
	default:
		log.Fatalf("migration: source inconnue %q (attendu : %s ou %s)",
			cfg.Migration.Source, source.NomWebdfd, source.NomHFSQL)
	}
	if err != nil {
		log.Fatalf("migration: %v", err)
	}
	log.Printf("migration: source %s", cfg.Migration.Source)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health.MakeHealthHandler(
		health.Checker{
			Name:  "postgres",
			Check: func(ctx context.Context) error { return pg.Ping(ctx) },
		},
	))
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version":   version,
			"buildTime": buildTime,
			"source":    cfg.Migration.Source,
			"uptime":    time.Since(startTime).String(),
		})
	})

	log.Printf("rex-sync démarré sur le port %d", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), mux))
}
