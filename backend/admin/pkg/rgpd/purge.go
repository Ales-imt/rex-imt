package rgpd

import (
	"back-rex-admin/pkg/rgpd/gen"
	mgen "back-rex-admin/pkg/rgpd/mariadb/gen"
	"back-rex-common/pkg/services"
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// anonymizeAfter : délai au-delà duquel les champs PII sont mis à NULL.
// deleteAfter    : délai au-delà duquel les enregistrements sont supprimés.
const (
	anonymizeAfter = 1 // an
	deleteAfter    = 3 // ans
)

// StartPurge lance la purge RGPD toutes les 24 h en arrière-plan.
// Une première exécution est déclenchée immédiatement au démarrage.
func StartPurge(pgCfg *services.DatabaseConfig, mariaCfg *services.MariaDBConfig) {
	pool := services.NewPG(context.Background(), services.ToDBS(pgCfg))

	mariaDb, err := services.NewMariaDBConnection(*mariaCfg)
	if err != nil {
		log.Printf("[rgpd] connexion MariaDB impossible, purge comptes sortis désactivée: %v", err)
	}

	go func() {
		runPurge(pool, mariaDb)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runPurge(pool, mariaDb)
		}
	}()
}

func runPurge(pool *services.Postgres, mariaDb *sql.DB) {
	ctx := context.Background()
	now := time.Now()
	log.Println("[rgpd] début de la purge")

	q := gen.New(pool.Db)
	anonymizeThreshold := now.AddDate(-anonymizeAfter, 0, 0)
	deleteThreshold := now.AddDate(-deleteAfter, 0, 0)

	anonymizeFeedback(ctx, q, anonymizeThreshold)
	anonymizeClassification(ctx, q, anonymizeThreshold)
	anonymizeEvalVerbatim(ctx, q, anonymizeThreshold)
	deleteClassification(ctx, q, deleteThreshold)
	deleteFeedback(ctx, q, deleteThreshold)

	if mariaDb != nil {
		purgeComptesSortis(ctx, q, mariaDb)
	}

	log.Println("[rgpd] purge terminée")
}

func ts(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func anonymizeFeedback(ctx context.Context, q *gen.Queries, threshold time.Time) {
	n, err := q.AnonymizeFeedback(ctx, ts(threshold))
	if err != nil {
		log.Printf("[rgpd] erreur anonymisation feedback: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[rgpd] %d feedback(s) anonymisé(s)", n)
	}
}

func anonymizeClassification(ctx context.Context, q *gen.Queries, threshold time.Time) {
	n, err := q.AnonymizeClassification(ctx, ts(threshold))
	if err != nil {
		log.Printf("[rgpd] erreur anonymisation feedback_classification: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[rgpd] %d classification(s) anonymisée(s)", n)
	}
}

func anonymizeEvalVerbatim(ctx context.Context, q *gen.Queries, threshold time.Time) {
	n, err := q.AnonymizeEvalVerbatim(ctx, ts(threshold))
	if err != nil {
		log.Printf("[rgpd] erreur anonymisation eval_verbatim: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[rgpd] %d verbatim(s) anonymisé(s)", n)
	}
}

func deleteClassification(ctx context.Context, q *gen.Queries, threshold time.Time) {
	n, err := q.DeleteClassification(ctx, ts(threshold))
	if err != nil {
		log.Printf("[rgpd] erreur suppression feedback_classification: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[rgpd] %d classification(s) supprimée(s)", n)
	}
}

func deleteFeedback(ctx context.Context, q *gen.Queries, threshold time.Time) {
	n, err := q.DeleteFeedback(ctx, ts(threshold))
	if err != nil {
		log.Printf("[rgpd] erreur suppression feedback: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[rgpd] %d feedback(s) supprimé(s) (cascade reponse + postit)", n)
	}
}

func purgeComptesSortis(ctx context.Context, q *gen.Queries, mariaDb *sql.DB) int {
	entrees, err := mgen.New(mariaDb).GetElevesSortis(ctx)
	if err != nil {
		log.Printf("[rgpd] erreur requête MariaDB comptes sortis: %v", err)
		return 0
	}

	if len(entrees) == 0 {
		log.Println("[rgpd] avertissement: aucun étudiant sorti trouvé dans Auréga (MariaDB vide ou base inaccessible ?)")
		return 0
	}

	count := 0
	for _, e := range entrees {
		datefin, ok := e.DerniereFin.(time.Time)
		if !ok {
			log.Printf("[rgpd] type inattendu pour datefin: email=%s", e.Mel.String)
			continue
		}

		userID, err := q.GetUserIDByEmail(ctx, e.Mel.String)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			log.Printf("[rgpd] erreur recherche compte: email=%s: %v", e.Mel.String, err)
			continue
		}

		if err := q.DeleteUserByID(ctx, userID); err != nil {
			log.Printf("[rgpd] erreur suppression compte: email=%s: %v", e.Mel.String, err)
			continue
		}

		log.Printf("[rgpd] RGPD purge compte: email=%s datefin=%s", e.Mel.String, datefin.Format("2006-01-02"))
		count++
	}

	if count > 0 {
		log.Printf("[rgpd] %d compte(s) purgé(s)", count)
	}

	return count
}
