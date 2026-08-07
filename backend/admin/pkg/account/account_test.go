package account

// Tests du cycle de vie des comptes, adossés à une vraie base (convention du
// dépôt : cf. pkg/migration/promotion_test.go — skip si la base est absente).
//
// L'enjeu couvert ici est l'INVARIANT du registre de présence : anonymiser un
// compte ne doit ni rompre la chaîne de hachage, ni faire disparaître un
// pointage. Les fixtures créent donc de vrais maillons via ledger.AppendLedger,
// la seule implémentation du chaînage.

import (
	"back-rex-common/pkg/ledger"
	"context"
	"fmt"
	"testing"
	"time"

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

// makeUser crée un utilisateur de test et programme sa suppression.
func makeUser(t *testing.T, db *pgxpool.Pool, suffix string) int32 {
	t.Helper()
	ctx := context.Background()
	email := fmt.Sprintf("test-account-%s-%d@example.invalid", suffix, time.Now().UnixNano())

	var id int32
	err := db.QueryRow(ctx,
		`INSERT INTO public."user" (name, surname, email, roles, auth_source)
		 VALUES ('Nom', 'Prenom', $1, ARRAY['ELEVE']::text[], 'ldap') RETURNING id`,
		email).Scan(&id)
	if err != nil {
		t.Fatalf("création utilisateur: %v", err)
	}

	t.Cleanup(func() {
		c := context.Background()
		// Les maillons de registre bloquent le DELETE (FK RESTRICT) : on les
		// retire d'abord — acceptable ICI seulement, en nettoyage de test.
		//
		// Ce retrait n'est sûr que parce qu'il défait la chaîne PAR LA FIN :
		// t.Cleanup s'exécute en ordre inverse des enregistrements, et les
		// maillons ont été ajoutés dans l'ordre de création des utilisateurs.
		// Retirer un maillon au MILIEU casserait le chaînage prev_hash des
		// suivants. Ne pas réordonner ces créations sans y repenser.
		db.Exec(c, `DELETE FROM presence_ledger WHERE user_id = $1`, id)
		db.Exec(c, `DELETE FROM public."user" WHERE id = $1`, id)
	})
	return id
}

// makeSeance crée une séance rattachée à une matière existante.
func makeSeance(t *testing.T, db *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()

	var matiereID int64
	if err := db.QueryRow(ctx, `SELECT id FROM matiere ORDER BY id LIMIT 1`).Scan(&matiereID); err != nil {
		t.Skipf("aucune matière en base: %v", err)
	}

	var id int64
	if err := db.QueryRow(ctx,
		`INSERT INTO seance (matiere_id) VALUES ($1) RETURNING id`, matiereID).Scan(&id); err != nil {
		t.Fatalf("création séance: %v", err)
	}
	t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM seance WHERE id = $1`, id) })
	return id
}

// givePresence pose un pointage ET un maillon de registre pour l'utilisateur.
func givePresence(t *testing.T, db *pgxpool.Pool, userID int32, seanceID int64) {
	t.Helper()
	ctx := context.Background()

	if _, err := db.Exec(ctx,
		`INSERT INTO pointage (seance_id, user_id, statut) VALUES ($1, $2, 'PRESENT')`,
		seanceID, userID); err != nil {
		t.Fatalf("création pointage: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, _, err := ledger.AppendLedger(ctx, tx, seanceID, userID, "PRESENT", time.Now()); err != nil {
		t.Fatalf("AppendLedger: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

type userRow struct {
	name, surname, email, authSource string
	disabled                         bool
}

func readUser(t *testing.T, db *pgxpool.Pool, id int32) userRow {
	t.Helper()
	var u userRow
	err := db.QueryRow(context.Background(),
		`SELECT name, surname, email, auth_source, disabled_at IS NOT NULL
		 FROM public."user" WHERE id = $1`, id).
		Scan(&u.name, &u.surname, &u.email, &u.authSource, &u.disabled)
	if err != nil {
		t.Fatalf("lecture utilisateur %d: %v", id, err)
	}
	return u
}

func countPointages(t *testing.T, db *pgxpool.Pool, userID int32) int {
	t.Helper()
	var n int
	if err := db.QueryRow(context.Background(),
		`SELECT count(*) FROM pointage WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("comptage pointages: %v", err)
	}
	return n
}

func userExists(t *testing.T, db *pgxpool.Pool, id int32) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM public."user" WHERE id = $1)`, id).Scan(&exists); err != nil {
		t.Fatalf("existence utilisateur: %v", err)
	}
	return exists
}

// TestAnonymizePreservesLedger — l'exigence centrale : après anonymisation d'un
// compte porteur de présence, la chaîne reste vérifiable et la ligne user ne
// contient plus aucune donnée nominative.
func TestAnonymizePreservesLedger(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	userID := makeUser(t, db, "anon")
	seanceID := makeSeance(t, db)
	givePresence(t, db, userID, seanceID)

	before, err := ledger.VerifyChain(ctx, db)
	if err != nil {
		t.Fatalf("VerifyChain avant: %v", err)
	}
	if !before.OK {
		t.Skipf("chaîne déjà rompue avant le test (seq %d): %s", before.BrokenAt, before.Error)
	}

	changed, err := Anonymize(ctx, db, userID)
	if err != nil {
		t.Fatalf("Anonymize: %v", err)
	}
	if !changed {
		t.Fatal("Anonymize aurait dû modifier le compte")
	}

	after, err := ledger.VerifyChain(ctx, db)
	if err != nil {
		t.Fatalf("VerifyChain après: %v", err)
	}
	if !after.OK {
		t.Fatalf("ANONYMISATION A ROMPU LA CHAÎNE (seq %d): %s", after.BrokenAt, after.Error)
	}

	u := readUser(t, db, userID)
	if u.name != "" || u.surname != "" {
		t.Errorf("identité résiduelle: name=%q surname=%q", u.name, u.surname)
	}
	if want := fmt.Sprintf("anonymise-%d@invalid.local", userID); u.email != want {
		t.Errorf("email = %q, attendu %q", u.email, want)
	}
	if u.authSource != "anonymized" {
		t.Errorf("auth_source = %q, attendu \"anonymized\"", u.authSource)
	}
	if !u.disabled {
		t.Error("un compte anonymisé doit être désactivé")
	}
}

// TestAnonymizeKeepsPointages — l'anonymisation ne doit déclencher aucune
// cascade : les pointages de l'utilisateur survivent.
func TestAnonymizeKeepsPointages(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	userID := makeUser(t, db, "pointage")
	seanceID := makeSeance(t, db)
	givePresence(t, db, userID, seanceID)

	if got := countPointages(t, db, userID); got != 1 {
		t.Fatalf("pointages avant = %d, attendu 1", got)
	}

	if _, err := Anonymize(ctx, db, userID); err != nil {
		t.Fatalf("Anonymize: %v", err)
	}

	if got := countPointages(t, db, userID); got != 1 {
		t.Errorf("pointages après anonymisation = %d, attendu 1 (cascade indue ?)", got)
	}
}

// TestIdempotence — un second appel ne doit affecter aucune ligne.
func TestIdempotence(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	t.Run("Disable", func(t *testing.T) {
		userID := makeUser(t, db, "disable-idem")

		changed, err := Disable(ctx, db, userID)
		if err != nil || !changed {
			t.Fatalf("1er Disable: changed=%v err=%v", changed, err)
		}
		changed, err = Disable(ctx, db, userID)
		if err != nil {
			t.Fatalf("2e Disable: %v", err)
		}
		if changed {
			t.Error("2e Disable aurait dû n'affecter aucune ligne")
		}
	})

	t.Run("Anonymize", func(t *testing.T) {
		userID := makeUser(t, db, "anon-idem")

		changed, err := Anonymize(ctx, db, userID)
		if err != nil || !changed {
			t.Fatalf("1er Anonymize: changed=%v err=%v", changed, err)
		}
		changed, err = Anonymize(ctx, db, userID)
		if err != nil {
			t.Fatalf("2e Anonymize: %v", err)
		}
		if changed {
			t.Error("2e Anonymize aurait dû n'affecter aucune ligne")
		}
	})
}

// lookupFixe simule Auréga avec une date de sortie connue.
func lookupFixe(d time.Time) SortieLookup {
	return func(context.Context, string) (time.Time, bool) { return d, true }
}

// lookupIndetermine simule Auréga indisponible ou étudiant inconnu.
func lookupIndetermine() SortieLookup {
	return func(context.Context, string) (time.Time, bool) { return time.Time{}, false }
}

// TestResolve couvre les trois branches de la décision CRUD.
func TestResolve(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	now := time.Now()
	const horizon = 10

	t.Run("sans présence → supprimé", func(t *testing.T) {
		userID := makeUser(t, db, "crud-del")

		d, err := Resolve(ctx, db, db, userID, lookupFixe(now.AddDate(-20, 0, 0)), now, horizon)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if d.Outcome != OutcomeDeleted {
			t.Fatalf("outcome = %q, attendu %q", d.Outcome, OutcomeDeleted)
		}
		if userExists(t, db, userID) {
			t.Error("le compte aurait dû être réellement supprimé")
		}
	})

	t.Run("présence + horizon non échu → conservé", func(t *testing.T) {
		userID := makeUser(t, db, "crud-keep")
		seanceID := makeSeance(t, db)
		givePresence(t, db, userID, seanceID)

		// Sortie récente : l'horizon de 10 ans est loin d'être atteint.
		d, err := Resolve(ctx, db, db, userID, lookupFixe(now.AddDate(-1, 0, 0)), now, horizon)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if d.Outcome != OutcomeDisabled {
			t.Fatalf("outcome = %q, attendu %q", d.Outcome, OutcomeDisabled)
		}

		u := readUser(t, db, userID)
		if u.authSource == "anonymized" {
			t.Error("le compte ne devait PAS être anonymisé avant l'échéance")
		}
		if u.name == "" {
			t.Error("l'identité doit être conservée pendant l'horizon")
		}
		if !u.disabled {
			t.Error("le compte devait être désactivé")
		}
	})

	t.Run("présence + horizon échu → anonymisé", func(t *testing.T) {
		userID := makeUser(t, db, "crud-anon")
		seanceID := makeSeance(t, db)
		givePresence(t, db, userID, seanceID)

		d, err := Resolve(ctx, db, db, userID, lookupFixe(now.AddDate(-11, 0, 0)), now, horizon)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if d.Outcome != OutcomeAnonymized {
			t.Fatalf("outcome = %q, attendu %q", d.Outcome, OutcomeAnonymized)
		}
		if u := readUser(t, db, userID); u.authSource != "anonymized" || u.name != "" {
			t.Errorf("compte non anonymisé: %+v", u)
		}
	})

	t.Run("présence + date indéterminée → conservé (fail-safe)", func(t *testing.T) {
		userID := makeUser(t, db, "crud-unknown")
		seanceID := makeSeance(t, db)
		givePresence(t, db, userID, seanceID)

		d, err := Resolve(ctx, db, db, userID, lookupIndetermine(), now, horizon)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if d.Outcome != OutcomeDisabled {
			t.Fatalf("outcome = %q, attendu %q — dans le doute on conserve", d.Outcome, OutcomeDisabled)
		}
		if u := readUser(t, db, userID); u.authSource == "anonymized" {
			t.Error("une date indéterminée ne doit JAMAIS déclencher l'anonymisation")
		}
	})
}

// TestResolveBulkMixte — un lot hétérogène : chaque compte reçoit son propre
// traitement, aucun n'est bloqué par le sort d'un autre.
func TestResolveBulkMixte(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	now := time.Now()
	const horizon = 10

	sansPresence := makeUser(t, db, "bulk-del")

	avecPresenceRecent := makeUser(t, db, "bulk-keep")
	givePresence(t, db, avecPresenceRecent, makeSeance(t, db))

	avecPresenceAncien := makeUser(t, db, "bulk-anon")
	givePresence(t, db, avecPresenceAncien, makeSeance(t, db))

	lookups := map[int32]time.Time{
		sansPresence:       now.AddDate(-2, 0, 0),
		avecPresenceRecent: now.AddDate(-2, 0, 0),
		avecPresenceAncien: now.AddDate(-11, 0, 0),
	}
	// La résolution passe par l'email : on retrouve l'id via une table inverse.
	emails := map[string]int32{}
	for id := range lookups {
		emails[readUser(t, db, id).email] = id
	}
	lookup := func(_ context.Context, email string) (time.Time, bool) {
		id, ok := emails[email]
		if !ok {
			return time.Time{}, false
		}
		return lookups[id], true
	}

	want := map[int32]Outcome{
		sansPresence:       OutcomeDeleted,
		avecPresenceRecent: OutcomeDisabled,
		avecPresenceAncien: OutcomeAnonymized,
	}

	for _, id := range []int32{sansPresence, avecPresenceRecent, avecPresenceAncien} {
		d, err := Resolve(ctx, db, db, id, lookup, now, horizon)
		if err != nil {
			t.Fatalf("Resolve(%d): %v", id, err)
		}
		if d.Outcome != want[id] {
			t.Errorf("compte %d: outcome = %q, attendu %q", id, d.Outcome, want[id])
		}
	}

	res, err := ledger.VerifyChain(ctx, db)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !res.OK {
		t.Fatalf("chaîne rompue après traitement du lot (seq %d): %s", res.BrokenAt, res.Error)
	}
}
