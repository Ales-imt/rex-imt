package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"filippo.io/age"
	"filippo.io/age/armor"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

type config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
	} `yaml:"database"`
	Age struct {
		PublicKey string `yaml:"publicKey"`
	} `yaml:"age"`
}

func loadConfig(path string) (*config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(raw))), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	var (
		configPath = flag.String("config", "config-local.yaml", "chemin du fichier de configuration")
		batchSize  = flag.Int("batch", 500, "taille de lot")
		dryRun     = flag.Bool("dry-run", false, "n'écrit rien en base")
	)
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("lecture config: %v", err)
	}

	recipient, err := age.ParseX25519Recipient(cfg.Age.PublicKey)
	if err != nil {
		log.Fatalf("parsing clé publique age: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.User, cfg.Database.Password, cfg.Database.Name)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("connexion BDD: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	for {
		n, err := migrerLot(ctx, db, recipient, *batchSize, *dryRun)
		if err != nil {
			log.Fatalf("migration: %v", err)
		}
		if n == 0 {
			break
		}
		log.Printf("lot traité: %d lignes", n)
	}

	log.Println("migration terminée")
}

func migrerLot(ctx context.Context, db *sql.DB, recipient age.Recipient, batch int, dryRun bool) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, user_id
		FROM feedback
		WHERE strongbox IS NULL AND user_id IS NOT NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batch)
	if err != nil {
		return 0, err
	}

	type ligne struct {
		id         int64
		etudiantID int64
	}
	var lignes []ligne
	for rows.Next() {
		var l ligne
		if err := rows.Scan(&l.id, &l.etudiantID); err != nil {
			rows.Close()
			return 0, err
		}
		lignes = append(lignes, l)
	}
	rows.Close()

	if len(lignes) == 0 {
		return 0, nil
	}

	for _, l := range lignes {
		payload := fmt.Sprintf("user_id=%d", l.etudiantID)
		var buf bytes.Buffer
		armorWriter := armor.NewWriter(&buf)
		w, err := age.Encrypt(armorWriter, recipient)
		if err != nil {
			return 0, fmt.Errorf("age.Encrypt ligne %d: %w", l.id, err)
		}
		if _, err := w.Write([]byte(payload)); err != nil {
			return 0, err
		}
		if err := w.Close(); err != nil {
			return 0, err
		}
		if err := armorWriter.Close(); err != nil {
			return 0, err
		}

		if dryRun {
			log.Printf("[dry-run] feedback id=%d →  strongbox=%d octets",
				l.id, buf.Len())
			continue
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE feedback
			SET  strongbox = $1
			WHERE id = $2
		`, buf.String(), l.id)
		if err != nil {
			return 0, fmt.Errorf("update ligne %d: %w", l.id, err)
		}
	}

	if dryRun {
		return len(lignes), nil
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(lignes), nil
}
