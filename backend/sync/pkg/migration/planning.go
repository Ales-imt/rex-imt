package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SyncPlanning charge le planning de chaque promo connue depuis la source
// amont, enrichit les noms de matières, et upserte groupes + matières.
//
// Ordre : 1. Récupérer les noms complets des matières. 2. Pour chaque promo,
// récupérer ses créneaux. 3. Déduplication puis upsert en base.
func SyncPlanning(ctx context.Context, src source.Source, db *pgxpool.Pool, ac anneeCourante) error {
	q := New(db)
	annee := ac.Annee

	// Relevé pris AVANT le premier UpsertSeanceMap : toute correspondance dont
	// le last_seen_at lui est antérieur n'a pas été revue pendant ce cycle.
	cycleStart := time.Now()

	// --- 1. Noms complets des matières ---
	coursNoms, err := src.CoursNoms()
	if err != nil {
		return err
	}

	// --- 2. Planning par promo ---
	promos, err := q.ListPromotionWebdfdIDs(ctx, annee)
	if err != nil {
		return err
	}

	// cocleToPromo: pour dériver la période, on veut savoir la promo de chaque COCLE.
	// On prend la première promo vue pour un COCLE donné (simplification).
	type cocleInfo struct {
		displayName string
		promoExtID  string
		promoID     int64
	}
	type grcleInfo struct {
		displayName string
		promoID     int64
	}

	cocles := make(map[string]cocleInfo)
	grcles := make(map[string]grcleInfo)

	// seanceKeys : clé "{cocle}_{date}_{hd}" → première occurrence de source.Creneau.
	// Correspond à l'unicité (matiere_id, starts_at) dans public.seance.
	seanceKeys := make(map[string]source.Creneau)

	startAnnee := ac.Debut.Format("20060102")
	endAnnee := ac.Fin.Format("20060102")

	// promosOK : promos dont le flux a été récupéré SANS erreur. C'est la seule
	// liste sur laquelle il est légitime d'annuler des séances — le `continue`
	// ci-dessous laisse volontairement les autres de côté (cf. seancesPerimees).
	promosOK := make(map[int64]bool, len(promos))

	for _, promo := range promos {
		entries, err := src.Planning(promo.ExternalID, startAnnee, endAnnee)
		if err != nil {
			log.Printf("planning: promo P0=%s inaccessible: %v — ignorée", promo.ExternalID, err)
			continue
		}
		promosOK[promo.InternalID] = true
		for _, e := range entries {
			if e.Cocle != "" && e.Cocle != "0" {
				if _, seen := cocles[e.Cocle]; !seen {
					cocles[e.Cocle] = cocleInfo{
						displayName: e.Cours,
						promoExtID:  e.P0cle,
						promoID:     promo.InternalID,
					}
				}
				if e.Plcle != "" {
					seanceKeys[e.Plcle] = e
				}
			}
			if e.Grcle != "" && e.Grcle != "0" {
				if _, seen := grcles[e.Grcle]; !seen {
					grcles[e.Grcle] = grcleInfo{
						displayName: e.Groupe,
						promoID:     promo.InternalID,
					}
				}
			}
		}
	}

	// --- 3. Année de chaque matière déduite de sa première séance ---
	// On ne peut pas se contenter de l'année courante : une matière est rattachée
	// à l'année (table public.annee) qui couvre la date de sa première séance.
	cocleFirstSeance := firstSeanceByCocle(seanceKeys)
	annees, err := q.ListAnnees(ctx)
	if err != nil {
		return fmt.Errorf("annee: liste: %w", err)
	}

	// --- 4. Upsert matières (construit cocleToMatiereID au passage) ---
	cocleToMatiereID := make(map[string]int64, len(cocles))
	mCount := 0
	for cocleStr, info := range cocles {
		nom := info.displayName
		if fullNom, ok := coursNoms[cocleStr]; ok {
			nom = fullNom
		}
		if nom == "" {
			continue
		}

		// Année rattachée à la première séance ; à défaut, année courante.
		matAnnee := annee
		if first, ok := cocleFirstSeance[cocleStr]; ok {
			if ay, found := anneeForDate(annees, first); found {
				matAnnee = ay
			}
		}

		matiereID, err := q.GetMatiereBySource(ctx, GetMatiereBySourceParams{Source: source.Map, ExternalID: cocleStr, Annee: annee})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("matiere_map: lookup planning COCLE=%s: %w", cocleStr, err)
		}
		if err == nil {
			if err = q.UpdateMatiereName(ctx, UpdateMatiereNameParams{Name: nom, ID: matiereID}); err != nil {
				return err
			}
			if err = q.UpdateMatiereAnnee(ctx, UpdateMatiereAnneeParams{Annee: matAnnee, ID: matiereID}); err != nil {
				return err
			}
		} else {
			matiereID, err = q.CreateMatiere(ctx, CreateMatiereParams{Name: nom, Annee: matAnnee})
			if err != nil {
				return err
			}
		}
		if err = q.UpsertMatiereMap(ctx, UpsertMatiereMapParams{
			InternalID: matiereID,
			Source:     source.Map,
			ExternalID: cocleStr,
			Annee:      annee,
		}); err != nil {
			return err
		}
		cocleToMatiereID[cocleStr] = matiereID

		// Période depuis le nom complet.
		pname, _ := semesterFromName(nom)
		periodeID, err := q.UpsertPeriode(ctx, UpsertPeriodeParams{Name: pname, PromotionID: info.promoID, Annee: annee})
		if err != nil {
			return err
		}
		if err = q.UpdateMatierePeriode(ctx, UpdateMatierePeriodeParams{
			PeriodeID: pgtype.Int8{Int64: periodeID, Valid: true},
			ID:        matiereID,
		}); err != nil {
			return err
		}
		mCount++
	}
	log.Printf("planning: %d matières synchronisées depuis le planning (annee=%d)", mCount, annee)

	// --- 5. Upsert groupes (construit grcleToGroupeID au passage) ---
	grcleToGroupeID := make(map[string]int64, len(grcles))
	gCount := 0
	for grcleStr, info := range grcles {
		groupeID, err := q.GetGroupeBySource(ctx, GetGroupeBySourceParams{Source: source.Map, ExternalID: grcleStr, Annee: annee})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("groupe_map: lookup planning GRCLE=%s: %w", grcleStr, err)
		}
		nom := info.displayName
		if nom == "" {
			nom = grcleStr
		}
		if err == nil {
			if err = q.UpdateGroupeName(ctx, UpdateGroupeNameParams{
				Name: pgtype.Text{String: nom, Valid: true},
				ID:   groupeID,
			}); err != nil {
				return err
			}
		} else {
			groupeID, err = q.CreateGroupe(ctx, CreateGroupeParams{
				Name:    pgtype.Text{String: nom, Valid: true},
				PromoID: info.promoID,
			})
			if err != nil {
				return err
			}
		}
		if err = q.UpsertGroupeMap(ctx, UpsertGroupeMapParams{
			InternalID: groupeID,
			Source:     source.Map,
			ExternalID: grcleStr,
			Annee:      annee,
		}); err != nil {
			return err
		}
		grcleToGroupeID[grcleStr] = groupeID
		gCount++
	}
	log.Printf("planning: %d groupes synchronisés depuis le planning (annee=%d)", gCount, annee)

	// --- 6. Résolution prof_id depuis prof_map ---
	prcleToUserID := make(map[string]int32)
	for _, e := range seanceKeys {
		if e.Prcle == "" {
			continue
		}
		if _, already := prcleToUserID[e.Prcle]; already {
			continue
		}
		uid, err := q.GetProfBySource(ctx, GetProfBySourceParams{Source: source.Map, ExternalID: e.Prcle})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("prof_map: lookup PRCLE=%s depuis planning: %w", e.Prcle, err)
		}
		if err == nil {
			prcleToUserID[e.Prcle] = uid
		}
	}

	// --- 7. Upsert séances ---
	sCount, ressuscitees, sErr := syncSeances(ctx, q, seanceKeys, cocleToMatiereID, grcleToGroupeID, prcleToUserID, promos)
	log.Printf("planning: %d séances synchronisées, %d rétablies (annee=%d)", sCount, ressuscitees, annee)
	if sErr != nil {
		return sErr
	}

	// --- 8. Séances disparues du planning amont ---
	// Après l'upsert : les créneaux encore présents viennent d'y rafraîchir leur
	// last_seen_at. `vus` porte TOUS les PL du flux, y compris ceux que
	// syncSeances a écartés (matière inconnue, horaire illisible) : ils existent
	// en amont, les annuler serait faux.
	vus := make(map[string]struct{}, len(seanceKeys))
	for plcle := range seanceKeys {
		vus[plcle] = struct{}{}
	}
	annulees, err := annulerSeancesDisparues(ctx, q, cycleStart, vus, promosOK, ac.Debut, ac.Fin)
	if err != nil {
		return fmt.Errorf("planning: annulation des séances disparues: %w", err)
	}
	log.Printf("planning: %d séances annulées (absentes du planning amont, annee=%d)", annulees, annee)
	return nil
}

// parisLoc est chargé une fois ; l'UTC est utilisé en fallback si la timezone est indisponible.
var parisLoc = func() *time.Location {
	l, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.UTC
	}
	return l
}()

const planningTimeFmt = "20060102 1504"

func parsePlanningTime(date, hhmm string) (time.Time, bool) {
	if len(date) != 8 || len(hhmm) != 4 {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(planningTimeFmt, date+" "+hhmm, parisLoc)
	return t, err == nil
}

// syncSeances synchronise chaque créneau amont (identifié par PL = plcle stable)
// dans public.seance + migration.seance_map.
// Schéma : lookup seance_map par PL → trouvé : UPDATE ; absent : INSERT.
//
// Retourne le nombre de séances traitées et, parmi elles, celles qui étaient
// annulées et que l'UPDATE vient de rétablir.
func syncSeances(
	ctx context.Context,
	q *Queries,
	seanceKeys map[string]source.Creneau,
	cocleToMatiereID map[string]int64,
	grcleToGroupeID map[string]int64,
	prcleToUserID map[string]int32,
	promos []ListPromotionWebdfdIDsRow,
) (int, int, error) {
	p0cleToPromoID := make(map[string]int64, len(promos))
	for _, p := range promos {
		p0cleToPromoID[p.ExternalID] = p.InternalID
	}

	var count, ressuscitees int
	for plcle, e := range seanceKeys {
		matiereID, ok := cocleToMatiereID[e.Cocle]
		if !ok {
			continue
		}
		startsAt, ok := parsePlanningTime(e.Date, e.HD)
		if !ok {
			continue
		}
		var endsAt pgtype.Timestamptz
		if t, ok2 := parsePlanningTime(e.Date, e.HF); ok2 {
			endsAt = pgtype.Timestamptz{Time: t, Valid: true}
		}
		var promoID pgtype.Int8
		if v := p0cleToPromoID[e.P0cle]; v != 0 {
			promoID = pgtype.Int8{Int64: v, Valid: true}
		}
		var groupeID pgtype.Int8
		if v := grcleToGroupeID[e.Grcle]; v != 0 {
			groupeID = pgtype.Int8{Int64: v, Valid: true}
		}
		salle := pgtype.Text{String: e.Salle, Valid: e.Salle != ""}
		prof := pgtype.Text{String: e.Prof, Valid: e.Prof != ""}
		var profID pgtype.Int4
		if uid, found := prcleToUserID[e.Prcle]; found {
			profID = pgtype.Int4{Int32: uid, Valid: true}
		}
		startsAtPg := pgtype.Timestamptz{Time: startsAt, Valid: true}

		seanceID, err := q.GetSeanceBySource(ctx, GetSeanceBySourceParams{Source: source.Map, ExternalID: plcle})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return count, ressuscitees, fmt.Errorf("seance_map: lookup PL=%s: %w", plcle, err)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			seanceID, err = q.CreateSeance(ctx, CreateSeanceParams{
				MatiereID:   matiereID,
				StartsAt:    startsAtPg,
				EndsAt:      endsAt,
				Salle:       salle,
				Prof:        prof,
				PromotionID: promoID,
				GroupeID:    groupeID,
				ProfID:      profID,
			})
		} else {
			var ressuscitee bool
			ressuscitee, err = q.UpdateSeance(ctx, UpdateSeanceParams{
				SeanceID:    seanceID,
				MatiereID:   matiereID,
				StartsAt:    startsAtPg,
				EndsAt:      endsAt,
				Salle:       salle,
				Prof:        prof,
				PromotionID: promoID,
				GroupeID:    groupeID,
				ProfID:      profID,
			})
			if ressuscitee {
				ressuscitees++
			}
		}
		if err != nil {
			return count, ressuscitees, err
		}
		// Rattrapage de la couverture des excuses : une séance créée (ou
		// déplacée) après la saisie d'une excuse tomberait sinon hors de sa
		// couverture, silencieusement. Idempotent (ON CONFLICT DO NOTHING).
		//
		// L'inverse n'est pas traité : une séance déplacée HORS d'une plage
		// couverte y reste rattachée — justification_seance est append-only, on
		// ne retire jamais une ligne. Le cas est marginal et la correction
		// passe par la modification de l'excuse, qui recalcule sa couverture.
		if err = q.AttacherJustificationsSeance(ctx, seanceID); err != nil {
			return count, ressuscitees, fmt.Errorf("justification_seance: séance %d: %w", seanceID, err)
		}
		if err = q.UpsertSeanceMap(ctx, UpsertSeanceMapParams{
			InternalID: seanceID,
			Source:     source.Map,
			ExternalID: plcle,
		}); err != nil {
			return count, ressuscitees, err
		}
		count++
	}
	return count, ressuscitees, nil
}

// firstSeanceByCocle parcourt les créneaux collectés et retourne, par COCLE
// (matière amont), l'horaire de début de la première séance rencontrée.
// Les créneaux dont la date/heure est illisible sont ignorés.
func firstSeanceByCocle(seanceKeys map[string]source.Creneau) map[string]time.Time {
	first := make(map[string]time.Time)
	for _, e := range seanceKeys {
		if e.Cocle == "" || e.Cocle == "0" {
			continue
		}
		t, ok := parsePlanningTime(e.Date, e.HD)
		if !ok {
			continue
		}
		if cur, seen := first[e.Cocle]; !seen || t.Before(cur) {
			first[e.Cocle] = t
		}
	}
	return first
}

// anneeForDate retourne l'année civile de début de l'année scolaire (table
// public.annee) dont la période [debut, fin] couvre la date de `t`. La
// comparaison se fait au jour près pour rester cohérente avec le type `date`
// des bornes en base. ok vaut false si aucune année ne couvre la date.
func anneeForDate(annees []ListAnneesRow, t time.Time) (int32, bool) {
	y, m, d := t.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	for _, a := range annees {
		debut := a.Debut.UTC()
		fin := a.Fin.UTC()
		debutDay := time.Date(debut.Year(), debut.Month(), debut.Day(), 0, 0, 0, 0, time.UTC)
		finDay := time.Date(fin.Year(), fin.Month(), fin.Day(), 0, 0, 0, 0, time.UTC)
		if !day.Before(debutDay) && !day.After(finDay) {
			return int32(a.Debut.Year()), true
		}
	}
	return 0, false
}

// semesterFromName extrait "S<n>" d'un nom commençant par "<n>.<x>.<y>…".
// Ex : "9.2.1 PROJET" → "S9". Retourne ("INCONNU", false) si absent.
func semesterFromName(nom string) (string, bool) {
	idx := strings.IndexByte(nom, '.')
	if idx < 1 {
		return "INCONNU", false
	}
	prefix := nom[:idx]
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return "INCONNU", false
		}
	}
	n, err := strconv.Atoi(prefix)
	if err != nil || n <= 0 {
		return "INCONNU", false
	}
	return fmt.Sprintf("S%d", n), true
}
