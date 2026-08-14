// Commande rex-export-webdfd : surveille le répertoire où sont déposées les
// sauvegardes HFSQL de cybema et convertit chaque archive en JSON.
//
// Le service ne connaît ni base de données ni réseau : il lit un répertoire,
// écrit un autre répertoire, et c'est tout. Cette pauvreté est voulue — il
// donne à mâcher des fichiers venus d'un partage à un exécutable WinDev sous
// wine, et n'a donc rien à faire d'un identifiant PostgreSQL.
//
// Sa sortie est reprise par rex-sync, selon le contrat décrit dans
// back-rex-common/pkg/hfsqlexport.
package main

import (
	"back-rex-common/pkg/services"
	"back-rex-export-webdfd/pkg/watcher"
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

// Injectées à la compilation.
var (
	buildTime string
	version   string
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	configPath := flag.String("config", "/opt/rex-export/conf/config.yaml", "fichier de configuration YAML")
	flag.Parse()

	log.Printf("rex-export-webdfd version: %s", version)
	log.Printf("Compilation time: %s", buildTime)

	cfg, err := services.LoadConfigYaml(*configPath)
	if err != nil {
		log.Fatal("Erreur chargement config YAML :", err)
	}

	// Arrêt propre : Close() sur le descripteur inotify débloque le Read() en
	// cours, ce qui fait sortir la boucle sans tuer un export en cours.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, unix.SIGTERM)
	defer stop()

	if err := watcher.Run(ctx, cfg.Migration.Watcher, nil); err != nil {
		log.Fatalf("Erreur : %v", err)
	}
}
