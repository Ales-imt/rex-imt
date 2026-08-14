package watcher

import (
	"back-rex-common/pkg/services"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// DefaultImage est l'image docker de l'export, a defaut de configuration.
const DefaultImage = "wine32-hf55"

// DefaultTimeout borne la duree d'un export docker, a defaut de configuration.
const DefaultTimeout = 15 * time.Minute

// Prepare valide la configuration du watcher et rend une copie completee de ses
// valeurs par defaut.
//
// La validation est faite au demarrage, pas au premier depot : une erreur de
// chemin ne doit pas se decouvrir la nuit ou tombe la sauvegarde.
func Prepare(cfg services.WatcherConfig) (services.WatcherConfig, error) {
	if cfg.Depot == "" {
		return cfg, errors.New("migration.watcher.depot obligatoire")
	}
	if !filepath.IsAbs(cfg.Depot) {
		return cfg, fmt.Errorf("migration.watcher.depot doit être un chemin absolu : %q", cfg.Depot)
	}
	if cfg.Data == "" {
		return cfg, errors.New("migration.watcher.data obligatoire")
	}
	if !filepath.IsAbs(cfg.Data) {
		return cfg, fmt.Errorf("migration.watcher.data doit être un chemin absolu : %q", cfg.Data)
	}

	if cfg.Runner == "" {
		cfg.Runner = ModeDocker
	}
	if cfg.Runner != ModeDocker && cfg.Runner != ModeLocal {
		return cfg, fmt.Errorf("migration.watcher.runner inconnu %q (attendu : %s ou %s)",
			cfg.Runner, ModeDocker, ModeLocal)
	}
	if cfg.Image == "" {
		cfg.Image = DefaultImage
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Timeout < 0 {
		return cfg, fmt.Errorf("migration.watcher.timeout négatif : %s", cfg.Timeout)
	}
	if cfg.Keep < 0 {
		return cfg, fmt.Errorf("migration.watcher.keep négatif : %d", cfg.Keep)
	}
	return cfg, nil
}
