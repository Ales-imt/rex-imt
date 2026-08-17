package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// syncPromotions upserte chaque promotion collectée dans public.promotion et
// insère la ligne de map correspondante dans migration.promotion_map.
func syncPromotions(ctx context.Context, q *Queries, promos []source.Promo, ac anneeCourante) error {
	annee := ac.Annee
	anneeID := ac.ID

	if err := q.CreateInconnuPromotion(ctx); err != nil {
		return err
	}

	var count int
	for _, p := range promos {
		promotionID, err := q.GetPromotionBySource(ctx, GetPromotionBySourceParams{
			Source:     source.Map,
			ExternalID: p.ExternalID,
			Annee:      annee,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("promotion_map: lookup P0=%s: %w", p.ExternalID, err)
		}
		if err == nil {
			if err = q.UpdatePromotionName(ctx, UpdatePromotionNameParams{
				Name: pgtype.Text{String: p.Nom, Valid: true},
				ID:   promotionID,
			}); err != nil {
				return err
			}
		} else {
			promotionID, err = q.CreatePromotion(ctx, pgtype.Text{String: p.Nom, Valid: true})
			if err != nil {
				return err
			}
		}
		if err = q.UpsertPromotionMap(ctx, UpsertPromotionMapParams{
			InternalID: promotionID,
			Source:     source.Map,
			ExternalID: p.ExternalID,
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
