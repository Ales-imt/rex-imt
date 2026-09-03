package presence

// Effectif figé à la clôture (public.seance_effectif + vue
// seance_effectif_resolu).
//
// Ce que ces tests protègent : une feuille de présence est une pièce
// justificative. Tant que l'effectif attendu était recalculé depuis
// eleve_groupe — un état COURANT, réconcilié à chaque synchronisation — un
// simple changement de groupe réécrivait les feuilles PASSÉES de l'élève, et un
// PDF de semestre régénéré différait de celui déjà émis.
//
// Les scénarios ci-dessous s'appuient sur une vraie base : la règle vit
// entièrement en SQL (vue + INSERT SELECT transactionnel), elle ne peut pas
// être vérifiée par une fonction pure.

import (
	"back-rex-common/pkg/presencedata"
	presencegen "back-rex-common/pkg/presencedata/gen"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// fixtureEffectif crée une promotion complète avec deux groupes et une séance
// OUVERTE rattachée au premier groupe, plus un élève inscrit dans ce groupe.
type fixtureEff struct {
	seanceID          int64
	eleveID           int32
	autreID           int32
	groupeID          int64
	autreGroupeID     int64
	matiereID         int64
	promoID, periodID int64
}

func fixtureEffectif(t *testing.T, db *pgxpool.Pool) fixtureEff {
	t.Helper()
	ctx := context.Background()
	suffixe := time.Now().UnixNano()

	creerUser := func(role string) int32 {
		var id int32
		err := db.QueryRow(ctx,
			`INSERT INTO public."user" (name, surname, email, roles, auth_source)
			 VALUES ('Nom', 'Prenom', $1, ARRAY['ELEVE']::text[], 'ldap') RETURNING id`,
			fmt.Sprintf("test-effectif-%s-%d@example.invalid", role, suffixe)).Scan(&id)
		if err != nil {
			t.Fatalf("création utilisateur: %v", err)
		}
		t.Cleanup(func() { db.Exec(context.Background(), `DELETE FROM public."user" WHERE id = $1`, id) })
		return id
	}

	entier := func(sql string, args ...any) int64 {
		var id int64
		if err := db.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("fixture (%s): %v", sql, err)
		}
		return id
	}

	f := fixtureEff{eleveID: creerUser("eleve"), autreID: creerUser("autre")}
	f.promoID = entier(`INSERT INTO public.promotion (name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("promo-effectif-%d", suffixe))
	f.groupeID = entier(`INSERT INTO public.groupe (name, promo_id) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("groupe-effectif-a-%d", suffixe), f.promoID)
	f.autreGroupeID = entier(`INSERT INTO public.groupe (name, promo_id) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("groupe-effectif-b-%d", suffixe), f.promoID)
	f.periodID = entier(`INSERT INTO public.periode (name, promotion_id, annee) VALUES ($1, $2, 2026) RETURNING id`,
		fmt.Sprintf("periode-effectif-%d", suffixe), f.promoID)
	f.matiereID = entier(`INSERT INTO public.matiere (name, periode_id, annee) VALUES ($1, $2, 2026) RETURNING id`,
		fmt.Sprintf("matiere-effectif-%d", suffixe), f.periodID)

	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		f.eleveID, f.groupeID); err != nil {
		t.Fatalf("fixture eleve_groupe: %v", err)
	}

	// Séance encore OUVERTE : chaque test décide s'il la clôture, et quand.
	f.seanceID = entier(
		`INSERT INTO public.seance (matiere_id, starts_at, ends_at, promotion_id, groupe_id, opened_at)
		 VALUES ($1, '2026-05-19 09:00+02', '2026-05-19 11:00+02', $2, $3, '2026-05-19 09:00+02')
		 RETURNING id`, f.matiereID, f.promoID, f.groupeID)

	t.Cleanup(func() {
		c := context.Background()
		db.Exec(c, `DELETE FROM public.seance WHERE id = $1`, f.seanceID)
		db.Exec(c, `DELETE FROM public.eleve_groupe WHERE id_groupe = ANY($1)`, []int64{f.groupeID, f.autreGroupeID})
		db.Exec(c, `DELETE FROM public.matiere WHERE id = $1`, f.matiereID)
		db.Exec(c, `DELETE FROM public.periode WHERE id = $1`, f.periodID)
		db.Exec(c, `DELETE FROM public.groupe WHERE id = ANY($1)`, []int64{f.groupeID, f.autreGroupeID})
		db.Exec(c, `DELETE FROM public.promotion WHERE id = $1`, f.promoID)
	})

	return f
}

// idsPresence retourne les user_id de la feuille (effectif attendu), et ceux
// remontés en « hors groupe ».
func idsPresence(t *testing.T, db *pgxpool.Pool, seanceID int64) (attendus, horsGroupe []int32) {
	t.Helper()
	ctx := context.Background()
	q := presencegen.New(db)

	rows, err := q.ListPresence(ctx, seanceID)
	if err != nil {
		t.Fatalf("ListPresence: %v", err)
	}
	for _, r := range rows {
		attendus = append(attendus, r.UserID)
	}
	hg, err := q.ListPresenceHorsGroupe(ctx, seanceID)
	if err != nil {
		t.Fatalf("ListPresenceHorsGroupe: %v", err)
	}
	for _, r := range hg {
		horsGroupe = append(horsGroupe, r.UserID)
	}
	return attendus, horsGroupe
}

func contient(ids []int32, id int32) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// Le cœur du problème : une feuille clôturée ne doit plus bouger quand les
// groupes changent, ni par retrait ni par ajout.
func TestEffectifFigeResisteAuChangementDeGroupe(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()

	ferme, figes, err := presencedata.CloseSeanceEtFiger(ctx, db, f.seanceID)
	if err != nil {
		t.Fatalf("CloseSeanceEtFiger: %v", err)
	}
	if !ferme || figes != 1 {
		t.Fatalf("clôture = (%v, %d), attendu (true, 1)", ferme, figes)
	}

	// L'élève quitte le groupe de la séance, un autre y entre : exactement ce
	// que fait DeleteEleveGroupeAbsents à chaque synchronisation.
	if _, err := db.Exec(ctx, `DELETE FROM public.eleve_groupe WHERE num_etudiant = $1`, f.eleveID); err != nil {
		t.Fatalf("changement de groupe: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		f.autreID, f.groupeID); err != nil {
		t.Fatalf("ajout tardif au groupe: %v", err)
	}

	attendus, _ := idsPresence(t, db, f.seanceID)
	if !contient(attendus, f.eleveID) {
		t.Error("l'élève convoqué a disparu de la feuille après son changement de groupe")
	}
	if contient(attendus, f.autreID) {
		t.Error("un élève ajouté au groupe APRÈS la clôture apparaît sur la feuille")
	}
	if len(attendus) != 1 {
		t.Errorf("feuille de %d élèves, attendu 1 : %v", len(attendus), attendus)
	}
}

// Une séance non clôturée n'a pas encore de pièce à protéger : elle doit suivre
// l'effectif vivant, sinon aucun ajout de dernière minute ne serait visible.
func TestSeanceNonClotureeSuitEffectifVivant(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		f.autreID, f.groupeID); err != nil {
		t.Fatalf("ajout au groupe: %v", err)
	}

	attendus, _ := idsPresence(t, db, f.seanceID)
	if !contient(attendus, f.autreID) || len(attendus) != 2 {
		t.Errorf("feuille = %v, attendu les deux élèves du groupe", attendus)
	}
}

// Un élève qui a pointé sans faire partie de l'effectif reste hors groupe :
// « hors groupe » se définit par rapport à la vue, pas à eleve_groupe.
func TestPointageHorsEffectifResteHorsGroupe(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx,
		`INSERT INTO public.pointage (seance_id, user_id, statut) VALUES ($1, $2, 'PRESENT')`,
		f.seanceID, f.autreID); err != nil {
		t.Fatalf("pointage hors groupe: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), `DELETE FROM public.pointage WHERE seance_id = $1`, f.seanceID)
	})

	if _, _, err := presencedata.CloseSeanceEtFiger(ctx, db, f.seanceID); err != nil {
		t.Fatalf("CloseSeanceEtFiger: %v", err)
	}

	attendus, horsGroupe := idsPresence(t, db, f.seanceID)
	if contient(attendus, f.autreID) {
		t.Error("un élève hors groupe a été figé comme attendu")
	}
	if !contient(horsGroupe, f.autreID) {
		t.Errorf("hors groupe = %v, attendu l'élève ayant pointé sans être convoqué", horsGroupe)
	}
}

// La clôture est idempotente : un second appel ne doit ni dupliquer de lignes,
// ni réécrire figee_at avec l'effectif du jour.
func TestDoubleClotureNeRefigePas(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()

	if _, _, err := presencedata.CloseSeanceEtFiger(ctx, db, f.seanceID); err != nil {
		t.Fatalf("première clôture: %v", err)
	}
	var figeeAt time.Time
	if err := db.QueryRow(ctx,
		`SELECT figee_at FROM public.seance_effectif WHERE seance_id = $1 AND user_id = $2`,
		f.seanceID, f.eleveID).Scan(&figeeAt); err != nil {
		t.Fatalf("lecture figee_at: %v", err)
	}

	// Entre les deux clôtures, le groupe change : si le figement rejouait, la
	// feuille basculerait sur ce nouvel effectif.
	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		f.autreID, f.groupeID); err != nil {
		t.Fatalf("ajout au groupe: %v", err)
	}

	ferme, figes, err := presencedata.CloseSeanceEtFiger(ctx, db, f.seanceID)
	if err != nil {
		t.Fatalf("seconde clôture: %v", err)
	}
	if ferme || figes != 0 {
		t.Errorf("seconde clôture = (%v, %d), attendu (false, 0)", ferme, figes)
	}

	var nb int
	var figeeAt2 time.Time
	if err := db.QueryRow(ctx,
		`SELECT count(*), min(figee_at) FROM public.seance_effectif WHERE seance_id = $1`,
		f.seanceID).Scan(&nb, &figeeAt2); err != nil {
		t.Fatalf("relecture seance_effectif: %v", err)
	}
	if nb != 1 {
		t.Errorf("%d lignes figées, attendu 1", nb)
	}
	if !figeeAt2.Equal(figeeAt) {
		t.Errorf("figee_at réécrite : %s puis %s", figeeAt, figeeAt2)
	}
}

// Effectif calculable vide (séance sans groupe ni promotion, matière sans
// période…) : rien n'est figé. Une séance figée à zéro serait indistinguable
// d'une séance sans attendus et gèlerait une feuille vide pour toujours ;
// l'absence de lignes laisse la vue retomber sur sa branche vivante.
func TestEffectifVideNeFigeRien(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()

	// La séance perd sa cible et la matière sa période : plus aucune des trois
	// résolutions de la vue (groupe, promotion, repli par la matière) n'aboutit.
	if _, err := db.Exec(ctx,
		`UPDATE public.seance SET groupe_id = NULL, promotion_id = NULL WHERE id = $1`, f.seanceID); err != nil {
		t.Fatalf("détachement de la cible: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE public.matiere SET periode_id = NULL WHERE id = $1`, f.matiereID); err != nil {
		t.Fatalf("détachement de la période: %v", err)
	}

	ferme, figes, err := presencedata.CloseSeanceEtFiger(ctx, db, f.seanceID)
	if err != nil {
		t.Fatalf("CloseSeanceEtFiger: %v", err)
	}
	if !ferme || figes != 0 {
		t.Fatalf("clôture = (%v, %d), attendu (true, 0)", ferme, figes)
	}

	var nb int
	if err := db.QueryRow(ctx,
		`SELECT count(*) FROM public.seance_effectif WHERE seance_id = $1`, f.seanceID).Scan(&nb); err != nil {
		t.Fatalf("relecture seance_effectif: %v", err)
	}
	if nb != 0 {
		t.Errorf("%d lignes figées, attendu aucune", nb)
	}

	// Rétablie, la cible refait vivre la feuille : rien n'a été gelé à zéro.
	if _, err := db.Exec(ctx,
		`UPDATE public.seance SET groupe_id = $1, promotion_id = $2 WHERE id = $3`,
		f.groupeID, f.promoID, f.seanceID); err != nil {
		t.Fatalf("rétablissement de la cible: %v", err)
	}
	attendus, _ := idsPresence(t, db, f.seanceID)
	if len(attendus) != 1 {
		t.Errorf("feuille = %v, attendu l'élève du groupe", attendus)
	}
}

// La séance fait foi sur sa cible, pas la matière. Cas réel : une journée de
// rentrée mutualisée, dont la matière est rattachée à la promotion où la
// synchronisation a vu son COCLE en premier, ciblant un groupe d'une AUTRE
// promotion. Les élèves de ce groupe sont attendus ; ceux de la promotion de
// la matière ne le sont pas.
func TestEffectifSuitLeGroupeDeLaSeanceNonLaMatiere(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()
	suffixe := time.Now().UnixNano()

	// Une seconde promotion, avec son groupe et son élève : la cible réelle.
	var autrePromoID, autreGroupeID int64
	if err := db.QueryRow(ctx, `INSERT INTO public.promotion (name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("promo-mutualisee-%d", suffixe)).Scan(&autrePromoID); err != nil {
		t.Fatalf("promotion cible: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO public.groupe (name, promo_id) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("groupe-mutualise-%d", suffixe), autrePromoID).Scan(&autreGroupeID); err != nil {
		t.Fatalf("groupe cible: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		f.autreID, autreGroupeID); err != nil {
		t.Fatalf("inscription au groupe cible: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		db.Exec(c, `DELETE FROM public.eleve_groupe WHERE id_groupe = $1`, autreGroupeID)
		db.Exec(c, `DELETE FROM public.seance WHERE id = $1`, f.seanceID)
		db.Exec(c, `DELETE FROM public.groupe WHERE id = $1`, autreGroupeID)
		db.Exec(c, `DELETE FROM public.promotion WHERE id = $1`, autrePromoID)
	})

	// La séance garde la matière de la première promotion, mais vise le
	// groupe de la seconde.
	if _, err := db.Exec(ctx,
		`UPDATE public.seance SET promotion_id = $1, groupe_id = $2 WHERE id = $3`,
		autrePromoID, autreGroupeID, f.seanceID); err != nil {
		t.Fatalf("reciblage de la séance: %v", err)
	}

	attendus, _ := idsPresence(t, db, f.seanceID)
	if !contient(attendus, f.autreID) {
		t.Errorf("feuille = %v, l'élève du groupe ciblé par la séance manque", attendus)
	}
	if contient(attendus, f.eleveID) {
		t.Errorf("feuille = %v, un élève de la promotion de la matière est attendu à tort", attendus)
	}
}

// Séance de promotion entière (groupe_id NULL) : l'effectif est celui de
// seance.promotion_id, et non de la promotion de la matière.
func TestEffectifPromotionEntiereSuitLaSeance(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()
	suffixe := time.Now().UnixNano()

	var autrePromoID, autreGroupeID int64
	if err := db.QueryRow(ctx, `INSERT INTO public.promotion (name) VALUES ($1) RETURNING id`,
		fmt.Sprintf("promo-entiere-%d", suffixe)).Scan(&autrePromoID); err != nil {
		t.Fatalf("promotion cible: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO public.groupe (name, promo_id) VALUES ($1, $2) RETURNING id`,
		fmt.Sprintf("groupe-entier-%d", suffixe), autrePromoID).Scan(&autreGroupeID); err != nil {
		t.Fatalf("groupe cible: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2)`,
		f.autreID, autreGroupeID); err != nil {
		t.Fatalf("inscription au groupe cible: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		db.Exec(c, `DELETE FROM public.eleve_groupe WHERE id_groupe = $1`, autreGroupeID)
		db.Exec(c, `DELETE FROM public.seance WHERE id = $1`, f.seanceID)
		db.Exec(c, `DELETE FROM public.groupe WHERE id = $1`, autreGroupeID)
		db.Exec(c, `DELETE FROM public.promotion WHERE id = $1`, autrePromoID)
	})

	if _, err := db.Exec(ctx,
		`UPDATE public.seance SET promotion_id = $1, groupe_id = NULL WHERE id = $2`,
		autrePromoID, f.seanceID); err != nil {
		t.Fatalf("reciblage de la séance: %v", err)
	}

	attendus, _ := idsPresence(t, db, f.seanceID)
	if !contient(attendus, f.autreID) || contient(attendus, f.eleveID) || len(attendus) != 1 {
		t.Errorf("feuille = %v, attendu le seul élève de la promotion de la séance", attendus)
	}
}

// Séance ouverte hors créneau planifié (presencedata.OpenSeance) : ni groupe
// ni promotion. La vue se replie alors sur la promotion de la matière, seul
// rattachement disponible — sans lui, personne ne serait jamais attendu.
func TestEffectifSansCibleSeReplieSurLaMatiere(t *testing.T) {
	db := openDB(t)
	f := fixtureEffectif(t, db)
	ctx := context.Background()

	if _, err := db.Exec(ctx,
		`UPDATE public.seance SET groupe_id = NULL, promotion_id = NULL WHERE id = $1`, f.seanceID); err != nil {
		t.Fatalf("détachement de la cible: %v", err)
	}

	attendus, _ := idsPresence(t, db, f.seanceID)
	if !contient(attendus, f.eleveID) || len(attendus) != 1 {
		t.Errorf("feuille = %v, attendu l'élève de la promotion de la matière", attendus)
	}
}
