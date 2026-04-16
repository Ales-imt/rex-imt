package season

import (
	"back-rex-common/pkg/services"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// GetOrCreateCurrentSeason récupère la saison actuelle ou en crée une nouvelle si aucune ne correspond
func GetOrCreateCurrentSeason(r *http.Request) (int, error) {
	now := time.Now()
	log.Printf("Debug: GetOrCreateCurrentSeason called at %s", now.Format(time.RFC3339))

	// Récupération du contexte Postgres et initialisation de sqlc
	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db) // sqlc génère cette fonction New(*pgxpool.Pool)

	// 🔍 Essayer de récupérer une saison existante
	seasonID, err := queries.GetCurrentSeason(context.Background(), services.ToPgTimestamptz(&now))
	if err == nil {
		return int(seasonID), nil
	}

	// 🆕 Aucune saison active → on la crée
	start := now.Truncate(24 * time.Hour)
	end := start.Add(7 * 24 * time.Hour)

	seasonID, err = queries.CreateSeason(context.Background(), CreateSeasonParams{
		StartDate: services.ToPgTimestamptz(&start),
		EndDate:   services.ToPgTimestamptz(&end),
	})
	if err != nil {
		return 0, fmt.Errorf("échec de création de la saison : %w", err)
	}

	return int(seasonID), nil
}

// GetSeasonEndDate récupère la date de fin d'une saison existante
func GetSeasonEndDate(r *http.Request, seasonID int) (time.Time, error) {
	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db)

	endDate, err := queries.GetSeasonEndDate(context.Background(), int32(seasonID))
	if err != nil {
		return time.Time{}, fmt.Errorf("échec de récupération de la date de fin : %w", err)
	}

	return endDate.Time, nil
}
