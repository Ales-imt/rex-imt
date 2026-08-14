package watcher

import (
	"back-rex-common/pkg/services"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// chargerConfigYAML écrit un fragment de configuration puis le relit par le
// chemin normal du service, pour vérifier le décodage effectif.
func chargerConfigYAML(t *testing.T, contenu string) *services.Config {
	t.Helper()
	name := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(name, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := services.LoadConfigYaml(name)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestPrepareDefauts(t *testing.T) {
	cfg, err := Prepare(services.WatcherConfig{Depot: "/depot", Data: "/data"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Image != DefaultImage {
		t.Fatalf("image %q, attendu %q", cfg.Image, DefaultImage)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Fatalf("timeout %s, attendu %s", cfg.Timeout, DefaultTimeout)
	}
	// Le mode par défaut ne suppose rien du déploiement : docker sur l'hôte,
	// comme le faisait le binaire d'origine.
	if cfg.Runner != ModeDocker {
		t.Fatalf("runner %q, attendu %q", cfg.Runner, ModeDocker)
	}
}

func TestPrepareValeursExplicites(t *testing.T) {
	cfg, err := Prepare(services.WatcherConfig{
		Depot:   "/depot",
		Data:    "/data",
		Image:   "wine32-hf55:test",
		Timeout: 2*time.Hour + 30*time.Minute,
		Keep:    5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Image != "wine32-hf55:test" {
		t.Fatalf("image %q", cfg.Image)
	}
	if cfg.Timeout != 2*time.Hour+30*time.Minute {
		t.Fatalf("timeout %s", cfg.Timeout)
	}
	if cfg.Keep != 5 {
		t.Fatalf("keep %d", cfg.Keep)
	}
}

func TestPrepareErreurs(t *testing.T) {
	tests := []struct {
		nom    string
		cfg    services.WatcherConfig
		erreur string
	}{
		{"depot manquant", services.WatcherConfig{Data: "/data"}, "depot obligatoire"},
		{"data manquant", services.WatcherConfig{Depot: "/depot"}, "data obligatoire"},
		{"depot relatif", services.WatcherConfig{Depot: "tt", Data: "/data"}, "chemin absolu"},
		{"data relatif", services.WatcherConfig{Depot: "/depot", Data: "data"}, "chemin absolu"},
		{"timeout négatif", services.WatcherConfig{Depot: "/depot", Data: "/data", Timeout: -time.Second}, "timeout négatif"},
		{"keep négatif", services.WatcherConfig{Depot: "/depot", Data: "/data", Keep: -1}, "keep négatif"},
		{"runner inconnu", services.WatcherConfig{Depot: "/depot", Data: "/data", Runner: "podman"}, "runner inconnu"},
	}

	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			_, err := Prepare(tt.cfg)
			if err == nil {
				t.Fatal("erreur attendue")
			}
			if !strings.Contains(err.Error(), tt.erreur) {
				t.Fatalf("erreur %v, attendu un message contenant %q", err, tt.erreur)
			}
		})
	}
}

// La configuration est désormais lue par yaml.v3 via services.Config : une
// durée écrite "15m" doit y arriver telle quelle, sans type intermédiaire.
func TestTimeoutDepuisYAML(t *testing.T) {
	cfg := chargerConfigYAML(t, `
migration:
  source: hfsql
  watcher:
    depot: /depot
    data: /data
    timeout: 2h30m
    keep: 3
`)
	if cfg.Migration.Source != "hfsql" {
		t.Fatalf("source %q", cfg.Migration.Source)
	}
	if cfg.Migration.Watcher.Timeout != 2*time.Hour+30*time.Minute {
		t.Fatalf("timeout %s", cfg.Migration.Watcher.Timeout)
	}
	if cfg.Migration.Watcher.Keep != 3 {
		t.Fatalf("keep %d", cfg.Migration.Watcher.Keep)
	}
}
