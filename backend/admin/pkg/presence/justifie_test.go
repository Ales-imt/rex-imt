package presence

// Parité du marquage « excusé » entre les deux services.
//
// GET /presence/seance/{id}/presence (ici) et GET /pointage/seance/{id}/presence
// (back-rex-eleve) lisent la MÊME requête, presencegen.ListPresence, mais
// composent chacun leur JSON. Une divergence de mapping ferait voir « Absent »
// en rouge au prof sur mobile pendant que le PDF officiel dirait « Excusé ».
//
// Les deux services étant deux modules Go distincts, la parité se vérifie par
// deux tests jumeaux adossés à la même fixture : celui-ci, et
// TestPointageExposeJustifie dans back-rex-eleve/pkg/pointage. Toute évolution
// du contrat doit toucher les deux.

import (
	presencegen "back-rex-common/pkg/presencedata/gen"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDSN = "host=10.20.1.4 port=5432 user=postgres password=root dbname=db_rex sslmode=disable"

func openDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	db, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	if err = db.Ping(ctx); err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

// fixtureExcuse crée une promotion complète, un élève absent et excusé sur une
// séance, et retourne (seanceID, userID).
func fixtureExcuse(t *testing.T, db *pgxpool.Pool) (int64, int32) {
	t.Helper()
	ctx := context.Background()
	suffixe := time.Now().UnixNano()

	var eleveID, gestID int32
	creerUser := func(role string) int32 {
		var id int32
		err := db.QueryRow(ctx,
			`INSERT INTO public."user" (name, surname, email, roles, auth_source)
			 VALUES ('Nom', 'Prenom', $1, ARRAY['ELEVE']::text[], 'ldap') RETURNING id`,
			fmt.Sprintf("test-parite-admin-%s-%d@example.invalid", role, suffixe)).Scan(&id)
		if err != nil {
			t.Fatalf("création utilisateur: %v", err)
		}
		t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM public."user" WHERE id = $1`, id) })
		return id
	}
	eleveID = creerUser("eleve")
	gestID = creerUser("gest")

	entier := func(sql string, args ...any) int64 {
		var id int64
		if err := db.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("fixture (%s): %v", sql, err)
		}
		return id
	}
	promoID := entier(`INSERT INTO public.promotion (name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("promo-parite-admin-%d", suffixe))
	groupeID := entier(`INSERT INTO public.groupe (name, promo_id) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("groupe-parite-admin-%d", suffixe), promoID)
	periodeID := entier(`INSERT INTO public.periode (name, promotion_id, annee) VALUES ($1, $2, 2026) RETURNING id`,
		fmt.Sprintf("periode-parite-admin-%d", suffixe), promoID)
	matiereID := entier(`INSERT INTO public.matiere (name, periode_id, annee) VALUES ($1, $2, 2026) RETURNING id`,
		fmt.Sprintf("matiere-parite-admin-%d", suffixe), periodeID)
	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		eleveID, groupeID); err != nil {
		t.Fatalf("fixture eleve_groupe: %v", err)
	}

	seanceID := entier(
		`INSERT INTO public.seance (matiere_id, starts_at, ends_at, promotion_id, groupe_id, opened_at, closed_at)
		 VALUES ($1, '2026-05-19 09:00+02', '2026-05-19 11:00+02', $2, $3, '2026-05-19 09:00+02', '2026-05-19 11:00+02')
		 RETURNING id`, matiereID, promoID, groupeID)

	justifID := entier(
		`INSERT INTO public.justification (user_id, periode, created_by)
		 VALUES ($1, tstzrange('2026-05-19 08:00+02', '2026-05-19 18:00+02'), $2) RETURNING id`,
		eleveID, gestID)
	if _, err := db.Exec(ctx,
		`INSERT INTO public.justification_seance (justification_id, seance_id) VALUES ($1, $2)`,
		justifID, seanceID); err != nil {
		t.Fatalf("fixture justification_seance: %v", err)
	}

	t.Cleanup(func() {
		c := context.Background()
		// Les tables de justification sont append-only : le trigger refuse le
		// DELETE à quiconque. session_replication_role le suspend le temps du
		// nettoyage — réservé aux tests, jamais au code applicatif.
		tx, err := db.Begin(c)
		if err != nil {
			t.Logf("nettoyage: %v", err)
			return
		}
		defer tx.Rollback(c)
		if _, err := tx.Exec(c, `SET LOCAL session_replication_role = replica`); err != nil {
			t.Logf("nettoyage (session_replication_role): %v", err)
			return
		}
		tx.Exec(c, `DELETE FROM public.justification_seance WHERE justification_id = $1`, justifID)
		tx.Exec(c, `DELETE FROM public.justification WHERE id = $1`, justifID)
		tx.Exec(c, `DELETE FROM public.seance WHERE id = $1`, seanceID)
		tx.Exec(c, `DELETE FROM public.eleve_groupe WHERE id_groupe = $1`, groupeID)
		tx.Exec(c, `DELETE FROM public.matiere WHERE id = $1`, matiereID)
		tx.Exec(c, `DELETE FROM public.periode WHERE id = $1`, periodeID)
		tx.Exec(c, `DELETE FROM public.groupe WHERE id = $1`, groupeID)
		tx.Exec(c, `DELETE FROM public.promotion WHERE id = $1`, promoID)
		if err := tx.Commit(c); err != nil {
			t.Logf("nettoyage: %v", err)
		}
	})

	return seanceID, eleveID
}

func TestPresenceExposeJustifie(t *testing.T) {
	db := openDB(t)
	seanceID, eleveID := fixtureExcuse(t, db)

	// 1. Source partagée : la requête commune voit bien l'excuse.
	rows, err := presencegen.New(db).ListPresence(context.Background(), seanceID)
	if err != nil {
		t.Fatalf("ListPresence: %v", err)
	}
	var attendu bool
	trouve := false
	for _, r := range rows {
		if r.UserID == eleveID {
			attendu, trouve = r.Justifie, true
		}
	}
	if !trouve {
		t.Fatal("l'élève n'apparaît pas dans ListPresence")
	}
	if !attendu {
		t.Fatal("ListPresence ne remonte pas l'excuse : fixture incorrecte")
	}

	// 2. Contrat HTTP du service admin. Le middleware d'authentification est
	// court-circuité : c'est le mapping de la réponse qui est testé, pas lui.
	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/presence/seance/%d/presence", seanceID), nil)
	ctx := context.WithValue(req.Context(), services.PgCtxKey2, &services.Postgres{Db: db})
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("seanceId", fmt.Sprintf("%d", seanceID))
	ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)

	rec := httptest.NewRecorder()
	GetPresenceHandler(rec, req.WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("statut HTTP %d : %s", rec.Code, rec.Body.String())
	}
	var resp presenceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("réponse illisible: %v", err)
	}

	for _, e := range resp.Eleves {
		if e.UserID != eleveID {
			continue
		}
		if e.Justifie != attendu {
			t.Errorf("justifie = %v côté HTTP, %v dans ListPresence : les deux services divergeraient",
				e.Justifie, attendu)
		}
		if e.Statut != "ABSENT" {
			t.Errorf("statut = %q, attendu ABSENT (l'excuse ne crée aucun pointage)", e.Statut)
		}
		if resp.Excuses != 1 || resp.Absents != 0 {
			t.Errorf("compteurs : absents=%d excuses=%d, attendu 0 et 1 (catégories disjointes)",
				resp.Absents, resp.Excuses)
		}
		return
	}
	t.Fatal("l'élève n'apparaît pas dans la réponse HTTP")
}
