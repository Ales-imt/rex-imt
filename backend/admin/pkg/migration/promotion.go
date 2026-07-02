package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// SyncPromotions charge la liste des promotions depuis webdfd (promos_txt),
// upserte chaque promotion dans public.promotion et insère la ligne de map
// correspondante dans migration.promotion_map.
func SyncPromotions(ctx context.Context, url string, db *pgxpool.Pool, ac anneeCourante) error {
	q := New(db)
	annee := ac.Annee
	anneeID := ac.ID

	resp, err := webdfdGet(url)
	if err != nil {
		return fmt.Errorf("webdfd: promos_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return err
	}

	if err = q.CreateInconnuPromotion(ctx); err != nil {
		return err
	}

	var count int
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		p0Str := kv["P0"]
		nom := strings.TrimSpace(kv["NOM"])
		if p0Str == "" || nom == "" {
			continue
		}
		if _, err = strconv.ParseInt(p0Str, 10, 64); err != nil {
			continue
		}

		promotionID, err := q.GetPromotionBySource(ctx, GetPromotionBySourceParams{
			Source:     "webdfd",
			ExternalID: p0Str,
			Annee:      annee,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("promotion_map: lookup webdfd P0=%s: %w", p0Str, err)
		}
		if err == nil {
			if err = q.UpdatePromotionName(ctx, UpdatePromotionNameParams{
				Name: pgtype.Text{String: nom, Valid: true},
				ID:   promotionID,
			}); err != nil {
				return err
			}
		} else {
			promotionID, err = q.CreatePromotion(ctx, pgtype.Text{String: nom, Valid: true})
			if err != nil {
				return err
			}
		}
		if err = q.UpsertPromotionMap(ctx, UpsertPromotionMapParams{
			InternalID: promotionID,
			Source:     "webdfd",
			ExternalID: p0Str,
			Annee:      annee,
		}); err != nil {
			return err
		}
		if err = q.UpdatePromotionAnnee(ctx, UpdatePromotionAnneeParams{
			AnneeID: pgtype.Int8{Int64: anneeID, Valid: true},
			ID:      promotionID,
		}); err != nil {
			return err
		}
		count++
	}
	log.Printf("promotion: %d promotions synchronisées (annee=%d)", count, annee)
	return nil
}

func parseKV(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
}
