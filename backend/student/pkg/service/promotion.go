package service

import (
	"back-rex-eleve/pkg/service/gen"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func syncPromotions(ctx context.Context, url string, db *pgxpool.Pool) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("webdfd: promos_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return err
	}

	q := gen.New(db)

	if err = q.UpsertPromotion(ctx, gen.UpsertPromotionParams{
		ID:   0,
		Name: pgtype.Text{String: "inconnu", Valid: true},
	}); err != nil {
		return fmt.Errorf("promotion: upsert inconnu: %w", err)
	}

	var count int
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseMatiereKV(line)
		p0Str := kv["P0"]
		nom := strings.TrimSpace(kv["NOM"])
		if p0Str == "" || nom == "" {
			continue
		}
		id, err := strconv.ParseInt(p0Str, 10, 64)
		if err != nil {
			continue
		}
		if err = q.UpsertPromotion(ctx, gen.UpsertPromotionParams{
			ID:   id,
			Name: pgtype.Text{String: nom, Valid: true},
		}); err != nil {
			return fmt.Errorf("promotion: upsert id=%d: %w", id, err)
		}
		count++
	}
	log.Printf("promotion: %d promotions synchronisées", count)
	return nil
}
