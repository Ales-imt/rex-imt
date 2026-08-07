package rgpd

import (
	"back-rex-admin/pkg/account"
	"back-rex-admin/pkg/rgpd/gen"
	mgen "back-rex-admin/pkg/rgpd/mariadb/gen"
	"back-rex-common/pkg/ledger"
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
	anonymizeAfter = 1  // an
	deleteAfter    = 3  // ans
	rejectedAfter  = 90 // jours : purge des feedbacks refusés (jamais publiés,
	// contenu conservé chiffré le temps d'une éventuelle contestation)
)

// Cycle de vie des comptes, en deux étapes ancrées sur la date de sortie lue
// dans Auréga. Distinct de anonymizeAfter ci-dessus, qui porte sur les
// feedbacks et n'a rien à voir avec les comptes.
const (
	// comptesDisableAfterYears : accès coupé un an après le départ. L'identité
	// est conservée — elle reste nécessaire pour rattacher les pièces de
	// présence à une personne pendant tout l'horizon de conservation.
	comptesDisableAfterYears = 1

	// comptesAnonymizeAfterYears : plafond conservateur de 10 ans, couvrant
	// l'obligation la plus longue applicable aux pièces justificatives de
	// formation (cofinancement européen FSE+). Voir docs/rgpd-dpo.md §7.
	comptesAnonymizeAfterYears = 10
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
	purgeRejectedFeedback(ctx, q, now.AddDate(0, 0, -rejectedAfter))
	purgeRejectedVerbatim(ctx, q, now.AddDate(0, 0, -rejectedAfter))

	if mariaDb != nil {
		purgeComptesADesactiver(ctx, pool, mariaDb, now)
		purgeComptesAAnonymiser(ctx, pool, mariaDb, now)
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

func purgeRejectedFeedback(ctx context.Context, q *gen.Queries, threshold time.Time) {
	n, err := q.PurgeRejectedFeedback(ctx, ts(threshold))
	if err != nil {
		log.Printf("[rgpd] erreur purge feedback refusés: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[rgpd] %d feedback(s) refusé(s) purgé(s)", n)
	}
}

func purgeRejectedVerbatim(ctx context.Context, q *gen.Queries, threshold time.Time) {
	n, err := q.PurgeRejectedVerbatim(ctx, ts(threshold))
	if err != nil {
		log.Printf("[rgpd] erreur purge verbatims refusés: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[rgpd] %d verbatim(s) refusé(s) purgé(s)", n)
	}
}

// comptesSortisAvant retourne les (userID, email, datefin) des comptes Postgres
// correspondant à des étudiants sortis d'Auréga avant le seuil donné.
type compteSorti struct {
	userID  int32
	email   string
	datefin time.Time
}

func comptesSortisAvant(ctx context.Context, q *gen.Queries, mariaDb *sql.DB, seuil time.Time) []compteSorti {
	entrees, err := mgen.New(mariaDb).GetElevesSortisAvant(ctx, sql.NullTime{Time: seuil, Valid: true})
	if err != nil {
		log.Printf("[rgpd] erreur requête MariaDB comptes sortis (seuil %s): %v",
			seuil.Format("2006-01-02"), err)
		return nil
	}

	out := make([]compteSorti, 0, len(entrees))
	for _, e := range entrees {
		datefin, ok := e.DerniereFin.(time.Time)
		if !ok {
			log.Printf("[rgpd] type inattendu pour datefin: email=%s", e.Mel.String)
			continue
		}

		userID, err := q.GetUserIDByEmail(ctx, e.Mel.String)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // sorti d'Auréga mais sans compte REX : rien à faire
		}
		if err != nil {
			log.Printf("[rgpd] erreur recherche compte: email=%s: %v", e.Mel.String, err)
			continue
		}

		out = append(out, compteSorti{userID: userID, email: e.Mel.String, datefin: datefin})
	}
	return out
}

// purgeComptesADesactiver — étape 1 du cycle de vie : un an après le départ,
// l'accès est coupé et les sessions révoquées. L'identité n'est PAS touchée :
// elle reste nécessaire pour rattacher les pièces de présence à une personne
// pendant tout l'horizon de conservation.
func purgeComptesADesactiver(ctx context.Context, pool *services.Postgres, mariaDb *sql.DB, now time.Time) int {
	seuil := now.AddDate(-comptesDisableAfterYears, 0, 0)
	comptes := comptesSortisAvant(ctx, gen.New(pool.Db), mariaDb, seuil)

	if len(comptes) == 0 {
		log.Println("[rgpd] aucun compte à désactiver (Auréga vide ou inaccessible ?)")
		return 0
	}

	count := 0
	for _, c := range comptes {
		changed, err := account.Disable(ctx, pool.Db, c.userID)
		if err != nil {
			log.Printf("[rgpd] erreur désactivation compte: email=%s: %v", c.email, err)
			continue
		}
		if changed {
			log.Printf("[rgpd] compte désactivé: email=%s datefin=%s", c.email, c.datefin.Format("2006-01-02"))
			count++
		}
	}

	if count > 0 {
		log.Printf("[rgpd] %d compte(s) désactivé(s)", count)
	}
	return count
}

// purgeComptesAAnonymiser — étape 2 du cycle de vie : à l'échéance de
// conservation des pièces de présence, l'identité est effacée EN PLACE. L'id
// entier est conservé, presence_ledger et pointage ne sont jamais touchés.
//
// Garde-fou : si au moins un compte a été anonymisé, la chaîne du registre est
// revérifiée. Une anonymisation ne peut pas la casser (elle ne touche aucune
// donnée hachée), mais le coût de la vérification est négligeable au regard de
// ce qu'un faux négatif coûterait sur une preuve d'assiduité opposable.
func purgeComptesAAnonymiser(ctx context.Context, pool *services.Postgres, mariaDb *sql.DB, now time.Time) int {
	seuil := now.AddDate(-comptesAnonymizeAfterYears, 0, 0)
	comptes := comptesSortisAvant(ctx, gen.New(pool.Db), mariaDb, seuil)

	count := 0
	for _, c := range comptes {
		changed, err := account.Anonymize(ctx, pool.Db, c.userID)
		if err != nil {
			log.Printf("[rgpd] erreur anonymisation compte: id=%d: %v", c.userID, err)
			continue
		}
		if changed {
			// L'email n'est pas journalisé : il vient d'être effacé.
			log.Printf("[rgpd] compte anonymisé: id=%d datefin=%s", c.userID, c.datefin.Format("2006-01-02"))
			count++
		}
	}

	if count == 0 {
		return 0
	}
	log.Printf("[rgpd] %d compte(s) anonymisé(s)", count)

	res, err := ledger.VerifyChain(ctx, pool.Db)
	if err != nil {
		log.Printf("[rgpd] ALERTE: vérification du registre de présence impossible après anonymisation: %v", err)
		return count
	}
	if !res.OK {
		log.Printf("[rgpd] ALERTE: registre de présence rompu après anonymisation (seq %d): %s",
			res.BrokenAt, res.Error)
		return count
	}
	log.Println("[rgpd] registre de présence vérifié après anonymisation: chaîne intacte")

	return count
}
