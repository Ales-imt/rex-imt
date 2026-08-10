package justification

// Fixtures des tests adossés à une vraie base (convention du dépôt :
// cf. pkg/account/account_test.go — skip si la base est absente).
//
// Chaque test construit sa propre chaîne promotion → période → matière →
// séance et son propre couple (élève, gestionnaire) : c'est la résolution de
// groupe complète de ListPresence qui est en jeu, une fixture partielle ne
// prouverait rien.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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

// fixture : un élève, un gestionnaire et une promotion complète, isolés.
type fixture struct {
	db        *pgxpool.Pool
	EleveID   int32
	GestID    int32
	PromoID   int64
	PeriodeID int64
	MatiereID int64
	GroupeID  int64
}

func newFixture(t *testing.T, db *pgxpool.Pool) *fixture {
	t.Helper()
	ctx := context.Background()
	f := &fixture{db: db}
	suffixe := time.Now().UnixNano()

	f.EleveID = makeUser(t, db, fmt.Sprintf("eleve-%d", suffixe))
	f.GestID = makeUser(t, db, fmt.Sprintf("gest-%d", suffixe))

	exec := func(sql string, args ...any) int64 {
		var id int64
		if err := db.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("fixture (%s): %v", sql, err)
		}
		return id
	}

	f.PromoID = exec(`INSERT INTO public.promotion (name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("promo-justif-%d", suffixe))
	f.GroupeID = exec(`INSERT INTO public.groupe (name, promo_id) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("groupe-justif-%d", suffixe), f.PromoID)
	f.PeriodeID = exec(`INSERT INTO public.periode (name, promotion_id, annee) VALUES ($1, $2, 2026) RETURNING id`,
		fmt.Sprintf("periode-justif-%d", suffixe), f.PromoID)
	f.MatiereID = exec(`INSERT INTO public.matiere (name, periode_id, annee) VALUES ($1, $2, 2026) RETURNING id`,
		fmt.Sprintf("matiere-justif-%d", suffixe), f.PeriodeID)

	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		f.EleveID, f.GroupeID); err != nil {
		t.Fatalf("fixture eleve_groupe: %v", err)
	}

	t.Cleanup(func() {
		c := context.Background()
		db.Exec(c, `DELETE FROM public.eleve_groupe WHERE id_groupe = $1`, f.GroupeID)
		db.Exec(c, `DELETE FROM public.matiere WHERE id = $1`, f.MatiereID)
		db.Exec(c, `DELETE FROM public.periode WHERE id = $1`, f.PeriodeID)
		db.Exec(c, `DELETE FROM public.groupe WHERE id = $1`, f.GroupeID)
		db.Exec(c, `DELETE FROM public.promotion WHERE id = $1`, f.PromoID)
	})
	return f
}

func makeUser(t *testing.T, db *pgxpool.Pool, suffixe string) int32 {
	t.Helper()
	ctx := context.Background()
	var id int32
	err := db.QueryRow(ctx,
		`INSERT INTO public."user" (name, surname, email, roles, auth_source)
		 VALUES ('Nom', 'Prenom', $1, ARRAY['ELEVE']::text[], 'ldap') RETURNING id`,
		fmt.Sprintf("test-justif-%s@example.invalid", suffixe)).Scan(&id)
	if err != nil {
		t.Fatalf("création utilisateur: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM public."user" WHERE id = $1`, id)
	})
	return id
}

// makeSeance crée une séance du groupe de la fixture sur un créneau donné,
// exprimé en heure de Paris.
func (f *fixture) makeSeance(t *testing.T, debut, fin string) int64 {
	t.Helper()
	ctx := context.Background()
	d, err := parseParis(debut)
	if err != nil {
		t.Fatalf("créneau %q: %v", debut, err)
	}
	fn, err := parseParis(fin)
	if err != nil {
		t.Fatalf("créneau %q: %v", fin, err)
	}

	var id int64
	err = f.db.QueryRow(ctx,
		`INSERT INTO public.seance (matiere_id, starts_at, ends_at, promotion_id, groupe_id, opened_at)
		 VALUES ($1, $2, $3, $4, $5, $2) RETURNING id`,
		f.MatiereID, d, fn, f.PromoID, f.GroupeID).Scan(&id)
	if err != nil {
		t.Fatalf("création séance: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		f.db.Exec(c, `DELETE FROM public.pointage WHERE seance_id = $1`, id)
		f.db.Exec(c, `DELETE FROM public.seance WHERE id = $1`, id)
	})
	return id
}

// pointer enregistre un pointage réel de l'élève sur une séance.
func (f *fixture) pointer(t *testing.T, seanceID int64, statut string) {
	t.Helper()
	if _, err := f.db.Exec(context.Background(),
		`INSERT INTO public.pointage (seance_id, user_id, statut) VALUES ($1, $2, $3)`,
		seanceID, f.EleveID, statut); err != nil {
		t.Fatalf("pointage: %v", err)
	}
}

// periodeDe construit la plage d'une excuse à partir de deux heures locales.
func periodeDe(t *testing.T, debut, fin string) pgtype.Range[pgtype.Timestamptz] {
	t.Helper()
	p, err := makePeriode(debut, fin)
	if err != nil {
		t.Fatalf("plage %s → %s : %v", debut, fin, err)
	}
	return p
}

// creerJustification reproduit la transaction du handler de création :
// insertion + matérialisation de la couverture. Les tests portent sur le
// comportement en base, pas sur le décodage HTTP.
func (f *fixture) creerJustification(t *testing.T, debut, fin string, seanceIDs []int64) int64 {
	t.Helper()
	ctx := context.Background()
	q := New(f.db)

	cree, err := q.CreateJustification(ctx, CreateJustificationParams{
		UserID:    f.EleveID,
		Periode:   periodeDe(t, debut, fin),
		CreatedBy: f.GestID,
	})
	if err != nil {
		t.Fatalf("création justification: %v", err)
	}
	f.nettoyerJustification(t, cree.ID)

	if seanceIDs == nil {
		seanceIDs = f.seancesCouvertes(t, debut, fin)
	}
	if len(seanceIDs) > 0 {
		if err := q.InsertJustificationSeances(ctx, InsertJustificationSeancesParams{
			JustificationID: cree.ID,
			SeanceIds:       seanceIDs,
		}); err != nil {
			t.Fatalf("couverture: %v", err)
		}
	}
	return cree.ID
}

func (f *fixture) seancesCouvertes(t *testing.T, debut, fin string) []int64 {
	t.Helper()
	rows, err := New(f.db).ListSeancesCouvertes(context.Background(), ListSeancesCouvertesParams{
		UserID:  f.EleveID,
		Periode: periodeDe(t, debut, fin),
	})
	if err != nil {
		t.Fatalf("ListSeancesCouvertes: %v", err)
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

func (f *fixture) revoquer(t *testing.T, id int64) {
	t.Helper()
	if err := New(f.db).RevokeJustification(context.Background(), RevokeJustificationParams{
		JustificationID: id,
		RevokedBy:       f.GestID,
	}); err != nil {
		t.Fatalf("révocation: %v", err)
	}
}

// nettoyerJustification programme la suppression d'une justification de test.
//
// Les trois tables sont append-only : le trigger deny_mutation refuse le DELETE
// à quiconque, propriétaire et superutilisateur compris — c'est précisément sa
// raison d'être. Le nettoyage passe donc par session_replication_role, qui
// suspend les triggers le temps d'une transaction. Réservé au nettoyage de
// test : aucun code applicatif ne doit emprunter ce chemin.
func (f *fixture) nettoyerJustification(t *testing.T, id int64) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		tx, err := f.db.Begin(ctx)
		if err != nil {
			t.Logf("nettoyage justification %d: %v", id, err)
			return
		}
		defer tx.Rollback(ctx)

		if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
			t.Logf("nettoyage justification %d (session_replication_role): %v", id, err)
			return
		}
		tx.Exec(ctx, `DELETE FROM public.justification_revocation WHERE justification_id = $1`, id)
		tx.Exec(ctx, `DELETE FROM public.justification_seance WHERE justification_id = $1`, id)
		tx.Exec(ctx, `UPDATE public.justification SET replaces_id = NULL WHERE replaces_id = $1`, id)
		tx.Exec(ctx, `DELETE FROM public.justification WHERE id = $1`, id)
		if err := tx.Commit(ctx); err != nil {
			t.Logf("nettoyage justification %d: %v", id, err)
		}
	})
}
