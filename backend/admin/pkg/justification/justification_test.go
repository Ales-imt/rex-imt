package justification

// Tests du comportement en base des excuses. L'invariant central : une excuse
// est une COUCHE posée sur l'absence — elle n'écrit jamais dans `pointage`, ne
// remplace jamais un statut de fait, et rien n'est jamais modifié ni supprimé.

import (
	presencegen "back-rex-common/pkg/presencedata/gen"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// presenceEleve retourne (statut, justifie) de l'élève de la fixture, tels que
// les lit la feuille de présence partagée par les deux services.
func (f *fixture) presenceEleve(t *testing.T, seanceID int64) (string, bool) {
	t.Helper()
	rows, err := presencegen.New(f.db).ListPresence(context.Background(), seanceID)
	if err != nil {
		t.Fatalf("ListPresence: %v", err)
	}
	n := 0
	var statut string
	var justifie bool
	for _, r := range rows {
		if r.UserID == f.EleveID {
			n++
			statut, justifie = r.Statut, r.Justifie
		}
	}
	if n != 1 {
		t.Fatalf("ListPresence: %d lignes pour l'élève %d, attendu exactement 1", n, f.EleveID)
	}
	return statut, justifie
}

func (f *fixture) modifier(t *testing.T, ancienID int64, debut, fin string) int64 {
	t.Helper()
	ctx := context.Background()
	q := New(f.db)

	f.revoquer(t, ancienID)
	cree, err := q.CreateJustification(ctx, CreateJustificationParams{
		UserID:     f.EleveID,
		Periode:    periodeDe(t, debut, fin),
		ReplacesID: pgtype.Int8{Int64: ancienID, Valid: true},
		CreatedBy:  f.GestID,
	})
	if err != nil {
		t.Fatalf("modification: %v", err)
	}
	f.nettoyerJustification(t, cree.ID)

	if ids := f.seancesCouvertes(t, debut, fin); len(ids) > 0 {
		if err := q.InsertJustificationSeances(ctx, InsertJustificationSeancesParams{
			JustificationID: cree.ID,
			SeanceIds:       ids,
		}); err != nil {
			t.Fatalf("couverture de la modification: %v", err)
		}
	}
	return cree.ID
}

// Une plage sans aucune séance reste enregistrable : le planning à venir peut
// encore la peupler (cf. AttacherJustificationsSeance).
func TestPlageSansSeance(t *testing.T) {
	f := newFixture(t, openDB(t))

	id := f.creerJustification(t, "2026-05-04T08:00", "2026-05-04T18:00", nil)

	row, err := New(f.db).GetJustification(context.Background(), id)
	if err != nil {
		t.Fatalf("GetJustification: %v", err)
	}
	if row.NbSeances != 0 {
		t.Errorf("NbSeances = %d, attendu 0", row.NbSeances)
	}
	if row.Statut != "ACTIVE" {
		t.Errorf("Statut = %q, attendu ACTIVE", row.Statut)
	}
}

func TestPlageAvecSeance(t *testing.T) {
	f := newFixture(t, openDB(t))
	seance := f.makeSeance(t, "2026-05-05T09:00", "2026-05-05T11:00")

	if statut, justifie := f.presenceEleve(t, seance); statut != "ABSENT" || justifie {
		t.Fatalf("avant excuse : statut=%q justifie=%v, attendu ABSENT/false", statut, justifie)
	}

	f.creerJustification(t, "2026-05-05T08:00", "2026-05-05T18:00", nil)

	statut, justifie := f.presenceEleve(t, seance)
	if statut != "ABSENT" || !justifie {
		t.Errorf("après excuse : statut=%q justifie=%v, attendu ABSENT/true", statut, justifie)
	}
}

// Un étudiant souffrant à partir de 10 h a bien manqué le cours de 9 h 45 :
// c'est le chevauchement (&&) qui fait foi, pas l'inclusion.
func TestChevauchementPartiel(t *testing.T) {
	f := newFixture(t, openDB(t))
	avant := f.makeSeance(t, "2026-05-06T09:45", "2026-05-06T11:15")
	apres := f.makeSeance(t, "2026-05-06T14:00", "2026-05-06T16:00")
	horsPlage := f.makeSeance(t, "2026-05-07T09:00", "2026-05-07T11:00")

	f.creerJustification(t, "2026-05-06T10:00", "2026-05-06T18:00", nil)

	for _, cas := range []struct {
		nom      string
		seance   int64
		justifie bool
	}{
		{"séance commencée avant le début de l'excuse", avant, true},
		{"séance entièrement dans la plage", apres, true},
		{"séance du lendemain", horsPlage, false},
	} {
		if _, justifie := f.presenceEleve(t, cas.seance); justifie != cas.justifie {
			t.Errorf("%s : justifie=%v, attendu %v", cas.nom, justifie, cas.justifie)
		}
	}
}

// Le pointage l'emporte toujours : un étudiant excusé qui a scanné reste
// PRESENT, et aucune ligne `pointage` n'est ajoutée par l'excuse.
func TestPointageLEmporteSurExcuse(t *testing.T) {
	f := newFixture(t, openDB(t))
	seance := f.makeSeance(t, "2026-05-08T09:00", "2026-05-08T11:00")
	f.pointer(t, seance, "PRESENT")

	var avant int64
	if err := f.db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM public.pointage WHERE seance_id = $1`, seance).Scan(&avant); err != nil {
		t.Fatalf("comptage pointage: %v", err)
	}

	f.creerJustification(t, "2026-05-08T08:00", "2026-05-08T18:00", nil)

	statut, justifie := f.presenceEleve(t, seance)
	if statut != "PRESENT" {
		t.Errorf("statut = %q, attendu PRESENT — l'excuse ne doit pas écraser un pointage", statut)
	}
	if !justifie {
		t.Errorf("justifie = false : le marquage doit rester porté, l'affichage seul l'ignore quand le statut n'est pas ABSENT")
	}

	var apres int64
	if err := f.db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM public.pointage WHERE seance_id = $1`, seance).Scan(&apres); err != nil {
		t.Fatalf("comptage pointage: %v", err)
	}
	if apres != avant {
		t.Errorf("pointage : %d lignes après, %d avant — une excuse ne doit rien y écrire", apres, avant)
	}
}

// Deux excuses actives qui se chevauchent ne doivent produire ni doublon sur la
// feuille de présence, ni silence dans l'aperçu.
func TestDeuxJustificationsChevauchantes(t *testing.T) {
	f := newFixture(t, openDB(t))
	seance := f.makeSeance(t, "2026-05-11T09:00", "2026-05-11T11:00")

	premiere := f.creerJustification(t, "2026-05-11T08:00", "2026-05-11T12:00", nil)
	f.creerJustification(t, "2026-05-11T10:00", "2026-05-12T18:00", nil)

	// presenceEleve échoue si l'élève apparaît plus d'une fois.
	if _, justifie := f.presenceEleve(t, seance); !justifie {
		t.Error("justifie = false alors que deux excuses actives couvrent la séance")
	}

	conflits, err := New(f.db).ListJustificationsChevauchantes(context.Background(),
		ListJustificationsChevauchantesParams{
			UserID:  f.EleveID,
			Periode: periodeDe(t, "2026-05-11T09:30", "2026-05-11T10:30"),
		})
	if err != nil {
		t.Fatalf("ListJustificationsChevauchantes: %v", err)
	}
	if len(conflits) != 2 {
		t.Fatalf("aperçu : %d chevauchement(s) signalé(s), attendu 2", len(conflits))
	}

	// exclude_id retire la justification en cours de modification, pas les autres.
	conflits, err = New(f.db).ListJustificationsChevauchantes(context.Background(),
		ListJustificationsChevauchantesParams{
			UserID:    f.EleveID,
			Periode:   periodeDe(t, "2026-05-11T09:30", "2026-05-11T10:30"),
			ExcludeID: pgtype.Int8{Int64: premiere, Valid: true},
		})
	if err != nil {
		t.Fatalf("ListJustificationsChevauchantes (exclude): %v", err)
	}
	if len(conflits) != 1 {
		t.Errorf("aperçu avec exclude_id : %d chevauchement(s), attendu 1", len(conflits))
	}
}

// Annuler rend les séances à « Absent » sans supprimer la moindre ligne.
func TestAnnulation(t *testing.T) {
	f := newFixture(t, openDB(t))
	seance := f.makeSeance(t, "2026-05-13T09:00", "2026-05-13T11:00")
	id := f.creerJustification(t, "2026-05-13T08:00", "2026-05-13T18:00", nil)

	if _, justifie := f.presenceEleve(t, seance); !justifie {
		t.Fatal("l'excuse n'est pas prise en compte avant annulation")
	}

	f.revoquer(t, id)

	statut, justifie := f.presenceEleve(t, seance)
	if statut != "ABSENT" || justifie {
		t.Errorf("après annulation : statut=%q justifie=%v, attendu ABSENT/false", statut, justifie)
	}

	ctx := context.Background()
	var nbJustif, nbSeances int64
	if err := f.db.QueryRow(ctx, `SELECT COUNT(*) FROM public.justification WHERE id = $1`, id).Scan(&nbJustif); err != nil {
		t.Fatalf("comptage justification: %v", err)
	}
	if err := f.db.QueryRow(ctx, `SELECT COUNT(*) FROM public.justification_seance WHERE justification_id = $1`, id).Scan(&nbSeances); err != nil {
		t.Fatalf("comptage justification_seance: %v", err)
	}
	if nbJustif != 1 || nbSeances != 1 {
		t.Errorf("après annulation : %d justification(s) et %d couverture(s) en base, attendu 1 et 1 — rien ne doit être supprimé",
			nbJustif, nbSeances)
	}

	row, err := New(f.db).GetJustification(ctx, id)
	if err != nil {
		t.Fatalf("GetJustification: %v", err)
	}
	if row.Statut != "ANNULEE" {
		t.Errorf("Statut = %q, attendu ANNULEE", row.Statut)
	}
}

// Élargir puis réduire la plage : la couverture suit dans les deux sens, et
// c'est toujours la dernière version qui fait foi.
func TestModificationElargitPuisReduit(t *testing.T) {
	f := newFixture(t, openDB(t))
	matin := f.makeSeance(t, "2026-05-18T09:00", "2026-05-18T11:00")
	apresMidi := f.makeSeance(t, "2026-05-18T14:00", "2026-05-18T16:00")

	id := f.creerJustification(t, "2026-05-18T08:00", "2026-05-18T12:00", nil)
	if _, justifie := f.presenceEleve(t, apresMidi); justifie {
		t.Fatal("la séance de l'après-midi ne devait pas être couverte")
	}

	elargie := f.modifier(t, id, "2026-05-18T08:00", "2026-05-18T18:00")
	if _, justifie := f.presenceEleve(t, apresMidi); !justifie {
		t.Error("après élargissement : la séance de l'après-midi devrait être couverte")
	}

	f.modifier(t, elargie, "2026-05-18T08:00", "2026-05-18T12:00")
	if _, justifie := f.presenceEleve(t, apresMidi); justifie {
		t.Error("après réduction : la séance de l'après-midi ne devrait plus être couverte")
	}
	if _, justifie := f.presenceEleve(t, matin); !justifie {
		t.Error("après réduction : la séance du matin devrait rester couverte")
	}
}

// Une chaîne de corrections doit rester lisible et ne laisser qu'une version
// active.
func TestChaineDeDeuxModifications(t *testing.T) {
	f := newFixture(t, openDB(t))
	ctx := context.Background()
	q := New(f.db)

	v1 := f.creerJustification(t, "2026-05-20T08:00", "2026-05-20T12:00", nil)
	v2 := f.modifier(t, v1, "2026-05-20T08:00", "2026-05-20T18:00")
	v3 := f.modifier(t, v2, "2026-05-20T08:00", "2026-05-21T18:00")

	for _, cas := range []struct {
		id             int64
		statut         string
		replaces       *int64
		attendReplaced bool
	}{
		{v1, "REMPLACEE", nil, true},
		{v2, "REMPLACEE", &v1, true},
		{v3, "ACTIVE", &v2, false},
	} {
		row, err := q.GetJustification(ctx, cas.id)
		if err != nil {
			t.Fatalf("GetJustification(%d): %v", cas.id, err)
		}
		if row.Statut != cas.statut {
			t.Errorf("justification %d : statut=%q, attendu %q", cas.id, row.Statut, cas.statut)
		}
		if cas.replaces == nil && row.ReplacesID.Valid {
			t.Errorf("justification %d : replaces_id=%d, attendu NULL", cas.id, row.ReplacesID.Int64)
		}
		if cas.replaces != nil && (!row.ReplacesID.Valid || row.ReplacesID.Int64 != *cas.replaces) {
			t.Errorf("justification %d : replaces_id=%+v, attendu %d", cas.id, row.ReplacesID, *cas.replaces)
		}
		if row.ReplacedByID.Valid != cas.attendReplaced {
			t.Errorf("justification %d : replaced_by présent=%v, attendu %v",
				cas.id, row.ReplacedByID.Valid, cas.attendReplaced)
		}
	}

	var actives int64
	if err := f.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM public.justification_active WHERE user_id = $1`, f.EleveID).Scan(&actives); err != nil {
		t.Fatalf("comptage actives: %v", err)
	}
	if actives != 1 {
		t.Errorf("%d justifications actives après deux corrections, attendu 1", actives)
	}
}

// 2026 comporte deux transitions d'heure. Une plage saisie « jusqu'à 18:00 »
// doit rester 18:00 heure de Paris de part et d'autre du 29 mars, pas 17:00 ni
// 19:00 : les deux séances de contrôle encadrent la borne à trente minutes.
func TestPlageAChevalSurLeChangementDHeure(t *testing.T) {
	f := newFixture(t, openDB(t))
	veille := f.makeSeance(t, "2026-03-28T17:00", "2026-03-28T18:00")
	jourJ := f.makeSeance(t, "2026-03-29T10:00", "2026-03-29T12:00")
	avantBorne := f.makeSeance(t, "2026-03-30T17:30", "2026-03-30T18:30")
	apresBorne := f.makeSeance(t, "2026-03-30T18:00", "2026-03-30T19:00")

	f.creerJustification(t, "2026-03-28T08:00", "2026-03-30T18:00", nil)

	for _, cas := range []struct {
		nom      string
		seance   int64
		justifie bool
	}{
		{"veille du changement d'heure (CET)", veille, true},
		{"jour du changement d'heure", jourJ, true},
		{"séance chevauchant la borne de fin (CEST)", avantBorne, true},
		{"séance commençant à la borne de fin", apresBorne, false},
	} {
		if _, justifie := f.presenceEleve(t, cas.seance); justifie != cas.justifie {
			t.Errorf("%s : justifie=%v, attendu %v", cas.nom, justifie, cas.justifie)
		}
	}
}

// Le verrou append-only est la seule garantie d'inaltérabilité de ces tables :
// il doit refuser UPDATE et DELETE sur les trois, y compris au propriétaire.
func TestTablesAppendOnly(t *testing.T) {
	f := newFixture(t, openDB(t))
	ctx := context.Background()
	seance := f.makeSeance(t, "2026-05-25T09:00", "2026-05-25T11:00")
	id := f.creerJustification(t, "2026-05-25T08:00", "2026-05-25T18:00", nil)
	f.revoquer(t, id)

	cas := []struct {
		nom string
		sql string
	}{
		{"UPDATE justification", `UPDATE public.justification SET user_id = user_id WHERE id = $1`},
		{"DELETE justification", `DELETE FROM public.justification WHERE id = $1`},
		{"UPDATE justification_seance", `UPDATE public.justification_seance SET seance_id = seance_id WHERE justification_id = $1`},
		{"DELETE justification_seance", `DELETE FROM public.justification_seance WHERE justification_id = $1`},
		{"UPDATE justification_revocation", `UPDATE public.justification_revocation SET revoked_by = revoked_by WHERE justification_id = $1`},
		{"DELETE justification_revocation", `DELETE FROM public.justification_revocation WHERE justification_id = $1`},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			// Chaque tentative dans sa propre transaction : la première erreur
			// avorterait les suivantes.
			tx, err := f.db.Begin(ctx)
			if err != nil {
				t.Fatalf("begin: %v", err)
			}
			defer tx.Rollback(ctx)
			if _, err := tx.Exec(ctx, c.sql, id); err == nil {
				t.Errorf("%s a réussi : la table n'est pas append-only", c.nom)
			}
		})
	}
	_ = seance
}
