// Package watcher surveille un repertoire de depot et lance l'export HFSQL
// vers JSON a chaque archive zip deposee.
//
// Equivalent Go de service/hfsql-depot-notify.sh, sans dependance a
// inotify-tools ni aux binaires unzip/zip : seul docker reste necessaire.
// Linux uniquement — inotify est appele directement via golang.org/x/sys/unix.
//
// Le paquet ne fait que produire : il depose sous <data> un repertoire
// d'export horodate, puis son marqueur de fin. C'est ce marqueur, et lui seul,
// qui signale a rex-sync qu'un instantane complet est lisible — le contrat est
// decrit dans back-rex-common/pkg/hfsqlexport.
package watcher

import (
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

// Run surveille le repertoire de depot jusqu'a l'annulation de ctx.
//
// onExport recoit le chemin du repertoire json de chaque export reussi. Il est
// appele en serie, depuis la boucle : deux exports docker simultanes se
// disputeraient les memes fichiers .FIC, et une synchronisation declenchee en
// parallele d'une autre lirait un export a moitie ecrit.
func Run(ctx context.Context, cfg services.WatcherConfig, onExport func(jsonDir string)) error {
	cfg, err := Prepare(cfg)
	if err != nil {
		return err
	}

	runner, err := NouveauRunner(cfg.Runner, cfg.Image, cfg.Entree)
	if err != nil {
		return err
	}
	exporter := &Exporter{
		Data:    cfg.Data,
		Timeout: cfg.Timeout,
		Keep:    cfg.Keep,
		Runner:  runner,
	}

	// IN_CLOSE_WRITE garantit que l'ecriture du fichier est terminee (un depot
	// par copie ne declenche donc qu'un seul export, en fin de copie) et
	// IN_MOVED_TO couvre le depot par renommage depuis le meme systeme de
	// fichiers.
	w, err := NewWatcher(cfg.Depot, unix.IN_CLOSE_WRITE|unix.IN_MOVED_TO)
	if err != nil {
		return err
	}
	defer w.Close()

	// Close() debloque le Read() en cours et fait sortir la boucle.
	arret := make(chan struct{})
	defer close(arret)
	go func() {
		select {
		case <-ctx.Done():
			log.Printf("watcher: arrêt demandé")
			w.Close()
		case <-arret:
		}
	}()

	log.Printf("watcher: surveillance de %s", cfg.Depot)
	for {
		events, err := w.Read()
		if err != nil {
			// Close() sur annulation du contexte : sortie normale.
			if errors.Is(err, os.ErrClosed) {
				return nil
			}
			return err
		}
		for _, ev := range events {
			log.Printf("watcher: déposé : %s", ev.Path)
			jsonDir, err := exporter.Process(ev.Path)
			if err != nil {
				// Un depot en echec ne doit pas arreter le service.
				log.Printf("watcher: erreur : %v", err)
				continue
			}
			// jsonDir vide : le fichier depose n'etait pas une archive.
			if jsonDir != "" && onExport != nil {
				onExport(jsonDir)
			}
		}
	}
}
