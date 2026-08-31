package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// connexionTest ouvre une transaction sur la base de test, annulée en fin de
// test : rien de ce que le test écrit ne survit, aucun nettoyage à prévoir.
func connexionTest(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testDSN)
	if err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { tx.Rollback(ctx) })
	return ctx, tx
}

func TestSyncSalles(t *testing.T) {
	ctx, tx := connexionTest(t)
	q := New(tx)

	salles := []source.Salle{
		{ExternalID: "999901", Nom: "TEST - AMPHI SALLES", Capacite: 120, Type: "Amphithéâtre"},
		// Capacité 0 et type vide : l'amont ne les renseigne pas, ils doivent
		// arriver NULL en base — pas 0, pas chaîne vide.
		{ExternalID: "999902", Nom: "TEST - FOND DE RESTAURANT", Capacite: 0, Type: ""},
	}

	res, err := syncSalles(ctx, q, salles, true)
	if err != nil {
		t.Fatalf("syncSalles: %v", err)
	}

	t.Run("public.salle et salle_map", func(t *testing.T) {
		var name string
		var capacite *int32
		var typ *string
		err := tx.QueryRow(ctx, `
			SELECT sa.name, sa.capacite, sa.type
			FROM public.salle sa
			JOIN migration.salle_map m ON m.internal_id = sa.id
			WHERE m.source = 'webdfd' AND m.external_id = '999901'`,
		).Scan(&name, &capacite, &typ)
		if err != nil {
			t.Fatalf("salle 999901 introuvable: %v", err)
		}
		if name != "TEST - AMPHI SALLES" || capacite == nil || *capacite != 120 || typ == nil || *typ != "Amphithéâtre" {
			t.Errorf("salle 999901 : name=%q capacite=%v type=%v", name, capacite, typ)
		}

		err = tx.QueryRow(ctx, `
			SELECT sa.name, sa.capacite, sa.type
			FROM public.salle sa
			JOIN migration.salle_map m ON m.internal_id = sa.id
			WHERE m.source = 'webdfd' AND m.external_id = '999902'`,
		).Scan(&name, &capacite, &typ)
		if err != nil {
			t.Fatalf("salle 999902 introuvable: %v", err)
		}
		if capacite != nil || typ != nil {
			t.Errorf("salle 999902 : capacite=%v type=%v, attendus NULL (0 amont = non renseigné)", capacite, typ)
		}
	})

	t.Run("résolveur", func(t *testing.T) {
		// SACLE connu : id + nom du RÉFÉRENTIEL, pas le libellé du créneau.
		id, nom := res.resoudre("999901", "libellé périmé du créneau")
		if !id.Valid || !nom.Valid || nom.String != "TEST - AMPHI SALLES" {
			t.Errorf("SACLE connu : id=%+v nom=%+v", id, nom)
		}
		// SACLE à 0 (distanciel, créneau sans salle) : repli sur le libellé.
		id, nom = res.resoudre("0", " B - CLAV ")
		if id.Valid || !nom.Valid || nom.String != "B - CLAV" {
			t.Errorf("SACLE=0 : id=%+v nom=%+v", id, nom)
		}
		// SACLE inconnu du référentiel : pas de rattachement, libellé conservé.
		id, nom = res.resoudre("888888", "SALLE FANTOME")
		if id.Valid || nom.String != "SALLE FANTOME" {
			t.Errorf("SACLE inconnu : id=%+v nom=%+v", id, nom)
		}
		// Ni clé ni libellé : les deux restent NULL.
		id, nom = res.resoudre("", "  ")
		if id.Valid || nom.Valid {
			t.Errorf("sans salle : id=%+v nom=%+v", id, nom)
		}
	})

	t.Run("second cycle : mise à jour, pas de doublon", func(t *testing.T) {
		maj := []source.Salle{
			{ExternalID: "999901", Nom: "TEST - AMPHI RENOMMÉ", Capacite: 90, Type: "Amphithéâtre"},
		}
		res2, err := syncSalles(ctx, q, maj, true)
		if err != nil {
			t.Fatalf("syncSalles (2e cycle): %v", err)
		}

		var n int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM migration.salle_map WHERE source = 'webdfd' AND external_id = '999901'`,
		).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%d entrées salle_map pour 999901, attendu 1", n)
		}
		if _, nom := res2.resoudre("999901", ""); nom.String != "TEST - AMPHI RENOMMÉ" {
			t.Errorf("nom après mise à jour : %q", nom.String)
		}
	})

	t.Run("amont muet : résolveur depuis la base", func(t *testing.T) {
		// ok=false — salles_txt n'a pas répondu. Rien n'est écrit, mais le
		// résolveur doit continuer à rattacher les salles déjà connues.
		res3, err := syncSalles(ctx, q, nil, false)
		if err != nil {
			t.Fatalf("syncSalles (amont muet): %v", err)
		}
		if id, _ := res3.resoudre("999902", ""); !id.Valid {
			t.Error("salle connue en base non rattachable quand l'amont ne répond pas")
		}
	})
}
