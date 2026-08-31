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
)

// syncPlanning écrit le planning collecté : matières, périodes, groupes,
// séances, puis annulation des créneaux disparus de l'amont.
//
// Rend la liste des séances dont la couverture d'excuses doit être revue
// (créées, rétablies, déplacées, ou dont l'effectif attendu a changé). Le
// rattachement lui-même est fait plus tard, une fois eleve_groupe à jour — cf.
// rattacherJustifications.
func syncPlanning(ctx context.Context, q *Queries, c *collecte, ac anneeCourante, cycleStart pgtype.Timestamptz, resSalle resolveurSalle) ([]int64, error) {
	annee := ac.Annee

	// Traduction des identifiants amont en identifiants internes : les
	// promotions viennent d'être écrites, leur map est donc à jour.
	promos, err := q.ListPromotionWebdfdIDs(ctx, annee)
	if err != nil {
		return nil, err
	}
	p0cleToPromoID := make(map[string]int64, len(promos))
	for _, p := range promos {
		p0cleToPromoID[p.ExternalID] = p.InternalID
	}

	// --- Matières et périodes ---
	cocleToMatiereID, err := syncMatieres(ctx, q, c, ac, p0cleToPromoID)
	if err != nil {
		return nil, err
	}

	// --- Groupes ---
	grcleToGroupeID, err := syncGroupes(ctx, q, c, annee, p0cleToPromoID)
	if err != nil {
		return nil, err
	}

	// --- Résolution prof_id depuis prof_map ---
	prcleToUserID := make(map[string]int32)
	for _, e := range c.creneaux {
		if e.Prcle == "" {
			continue
		}
		if _, already := prcleToUserID[e.Prcle]; already {
			continue
		}
		uid, err := q.GetProfBySource(ctx, GetProfBySourceParams{Source: source.Map, ExternalID: e.Prcle})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("prof_map: lookup PRCLE=%s depuis planning: %w", e.Prcle, err)
		}
		if err == nil {
			prcleToUserID[e.Prcle] = uid
		}
	}

	// --- Séances ---
	res, err := syncSeances(ctx, q, c, cocleToMatiereID, grcleToGroupeID, prcleToUserID, p0cleToPromoID, resSalle)
	log.Printf("planning: %d séances synchronisées, %d rétablies (annee=%d)", res.count, res.ressuscitees, annee)
	if res.ecartesMatiere > 0 || res.ecartesHoraire > 0 {
		log.Printf("planning: ATTENTION %d créneaux collectés n'ont pas produit de séance (%d matière inconnue, %d horaire illisible)",
			res.ecartesMatiere+res.ecartesHoraire, res.ecartesMatiere, res.ecartesHoraire)
	}
	// Ce compteur n'est pas un ornement : sans lui, un SACLE qui cesse d'être
	// servi en amont fait silencieusement chuter l'occupation de toutes les
	// salles à zéro, et l'écran affiche un établissement vide au lieu d'une
	// erreur.
	if res.sallesNonResolues > 0 {
		log.Printf("planning: %d séance(s) dont le SACLE ne résout pas — libellé amont conservé, hors bilan d'occupation", res.sallesNonResolues)
	}
	if err != nil {
		return res.aRattacher, err
	}

	// --- Séances disparues du planning amont ---
	// `vus` porte TOUS les PL du flux, y compris ceux que syncSeances a écartés
	// (matière inconnue, horaire illisible) : ils existent en amont, les
	// annuler serait faux.
	vus := make(map[string]struct{}, len(c.creneaux))
	for plcle := range c.creneaux {
		vus[plcle] = struct{}{}
	}
	// promosOK est indexé par P0 externe côté collecte ; seancesPerimees
	// compare à seance.promotion_id, donc en interne.
	promosOKInternes := make(map[int64]bool, len(c.promosOK))
	for p0 := range c.promosOK {
		if id, ok := p0cleToPromoID[p0]; ok {
			promosOKInternes[id] = true
		}
	}

	annulees, err := annulerSeancesDisparues(ctx, q, cycleStart, vus, promosOKInternes, ac.Debut, ac.Fin)
	if err != nil {
		return res.aRattacher, fmt.Errorf("planning: annulation des séances disparues: %w", err)
	}
	log.Printf("planning: %d séances annulées (absentes du planning amont, annee=%d)", annulees, annee)
	return res.aRattacher, nil
}

// syncMatieres upserte les matières du planning et leur période, et rend la
// table COCLE → matiere_id.
func syncMatieres(ctx context.Context, q *Queries, c *collecte, ac anneeCourante, p0cleToPromoID map[string]int64) (map[string]int64, error) {
	annee := ac.Annee

	// L'année d'une matière est celle qui couvre sa première séance, et non
	// mécaniquement l'année courante. En pratique les deux coïncident tant que
	// la fenêtre interrogée est [ac.Debut, ac.Fin] ; le calcul ne sert que si
	// une source rend des créneaux hors de la plage demandée.
	cocleFirstSeance := firstSeanceByCocle(c.creneaux)
	annees, err := q.ListAnnees(ctx)
	if err != nil {
		return nil, fmt.Errorf("annee: liste: %w", err)
	}

	cocleToMatiereID := make(map[string]int64, len(c.cocles))
	count := 0
	for cocleStr, info := range c.cocles {
		nom := info.displayName
		if fullNom, ok := c.coursNoms[cocleStr]; ok {
			nom = fullNom
		}
		if nom == "" {
			continue
		}

		matAnnee := annee
		if first, ok := cocleFirstSeance[cocleStr]; ok {
			if ay, found := anneeForDate(annees, first); found {
				matAnnee = ay
			}
		}

		matiereID, err := q.GetMatiereBySource(ctx, GetMatiereBySourceParams{Source: source.Map, ExternalID: cocleStr, Annee: annee})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("matiere_map: lookup planning COCLE=%s: %w", cocleStr, err)
		}
		if err == nil {
			if err = q.UpdateMatiereName(ctx, UpdateMatiereNameParams{Name: nom, ID: matiereID}); err != nil {
				return nil, err
			}
			if err = q.UpdateMatiereAnnee(ctx, UpdateMatiereAnneeParams{Annee: matAnnee, ID: matiereID}); err != nil {
				return nil, err
			}
		} else {
			matiereID, err = q.CreateMatiere(ctx, CreateMatiereParams{Name: nom, Annee: matAnnee})
			if err != nil {
				return nil, err
			}
		}
		if err = q.UpsertMatiereMap(ctx, UpsertMatiereMapParams{
			InternalID: matiereID,
			Source:     source.Map,
			ExternalID: cocleStr,
			Annee:      annee,
		}); err != nil {
			return nil, err
		}
		cocleToMatiereID[cocleStr] = matiereID

		// Période depuis le nom complet, rattachée à la promo où le COCLE a été
		// vu en premier. Pour un cours mutualisé, les autres promos n'auront
		// donc pas d'effectif attendu — la collecte l'a signalé.
		promoID, ok := p0cleToPromoID[info.promoExtID]
		if !ok {
			log.Printf("planning: COCLE=%s rattaché à la promo P0=%s inconnue en base, période ignorée", cocleStr, info.promoExtID)
			count++
			continue
		}
		pname, _ := semesterFromName(nom)
		periodeID, err := q.UpsertPeriode(ctx, UpsertPeriodeParams{Name: pname, PromotionID: promoID, Annee: annee})
		if err != nil {
			return nil, err
		}
		if err = q.UpdateMatierePeriode(ctx, UpdateMatierePeriodeParams{
			PeriodeID: pgtype.Int8{Int64: periodeID, Valid: true},
			ID:        matiereID,
		}); err != nil {
			return nil, err
		}
		count++
	}
	log.Printf("planning: %d matières synchronisées depuis le planning (annee=%d)", count, annee)
	return cocleToMatiereID, nil
}

// syncGroupes upserte les groupes du planning et rend la table GRCLE → groupe_id.
func syncGroupes(ctx context.Context, q *Queries, c *collecte, annee int32, p0cleToPromoID map[string]int64) (map[string]int64, error) {
	grcleToGroupeID := make(map[string]int64, len(c.grcles))
	count := 0
	for grcleStr, info := range c.grcles {
		promoID, ok := p0cleToPromoID[info.promoExtID]
		if !ok {
			log.Printf("planning: GRCLE=%s rattaché à la promo P0=%s inconnue en base, groupe ignoré", grcleStr, info.promoExtID)
			continue
		}

		groupeID, err := q.GetGroupeBySource(ctx, GetGroupeBySourceParams{Source: source.Map, ExternalID: grcleStr, Annee: annee})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("groupe_map: lookup planning GRCLE=%s: %w", grcleStr, err)
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
				return nil, err
			}
		} else {
			groupeID, err = q.CreateGroupe(ctx, CreateGroupeParams{
				Name:    pgtype.Text{String: nom, Valid: true},
				PromoID: promoID,
			})
			if err != nil {
				return nil, err
			}
		}
		if err = q.UpsertGroupeMap(ctx, UpsertGroupeMapParams{
			InternalID: groupeID,
			Source:     source.Map,
			ExternalID: grcleStr,
			Annee:      annee,
		}); err != nil {
			return nil, err
		}
		grcleToGroupeID[grcleStr] = groupeID
		count++
	}
	log.Printf("planning: %d groupes synchronisés depuis le planning (annee=%d)", count, annee)
	return grcleToGroupeID, nil
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

// resultatSeances porte le bilan d'un passage de syncSeances.
type resultatSeances struct {
	count        int
	ressuscitees int
	// aRattacher : séances dont la couverture d'excuses doit être recalculée.
	aRattacher []int64
	// Créneaux présents en amont mais non transformés en séance. Comptés même
	// sans dump : un écart entre les créneaux collectés et les séances écrites
	// doit toujours apparaître au journal, jamais seulement dans un fichier de
	// diagnostic qu'il faut penser à activer.
	ecartesMatiere int
	ecartesHoraire int
	// sallesNonResolues : séances dont le créneau annonce une salle que le
	// référentiel ne connaît pas. Écrites quand même, avec le libellé amont,
	// mais elles sortent du bilan d'occupation. Compté, pas ignoré.
	sallesNonResolues int
}

// syncSeances synchronise chaque créneau amont (identifié par PL = plcle stable)
// dans public.seance + migration.seance_map.
// Schéma : lookup seance_map par PL → trouvé : UPDATE ; absent : INSERT.
func syncSeances(
	ctx context.Context,
	q *Queries,
	c *collecte,
	cocleToMatiereID map[string]int64,
	grcleToGroupeID map[string]int64,
	prcleToUserID map[string]int32,
	p0cleToPromoID map[string]int64,
	resSalle resolveurSalle,
) (resultatSeances, error) {
	var res resultatSeances

	for plcle, e := range c.creneaux {
		matiereID, ok := cocleToMatiereID[e.Cocle]
		if !ok {
			// Le COCLE n'a pas produit de matière : nom vide côté amont, ou
			// promotion de rattachement inconnue (cf. syncMatieres).
			c.rejeter(rejetMatiereInconnue, e.P0cle, "COCLE="+e.Cocle, e)
			res.ecartesMatiere++
			continue
		}
		startsAt, ok := parsePlanningTime(e.Date, e.HD)
		if !ok {
			c.rejeter(rejetHoraireIllisble, e.P0cle, "date="+e.Date+" HD="+e.HD, e)
			res.ecartesHoraire++
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
		var profID pgtype.Int4
		if uid, found := prcleToUserID[e.Prcle]; found {
			profID = pgtype.Int4{Int32: uid, Valid: true}
		}
		// salle et salle_id sortent du MÊME appel au résolveur : le texte est le
		// name de la salle résolue par le SACLE, et ne retombe sur le libellé du
		// créneau que lorsque la clé ne résout pas. Les faire diverger rouvrirait
		// deux vocabulaires pour la même salle, selon l'écran.
		salleID, salle := resSalle.resoudre(e.Sacle, e.Salle)
		if !salleID.Valid && strings.TrimSpace(e.Salle) != "" {
			res.sallesNonResolues++
		}
		prof := pgtype.Text{String: e.Prof, Valid: e.Prof != ""}
		remarque := pgtype.Text{String: e.Note, Valid: e.Note != ""}
		startsAtPg := pgtype.Timestamptz{Time: startsAt, Valid: true}

		// grcleVide / prcleVide distinguent « l'amont dit qu'il n'y a pas de
		// groupe / de prof » de « on n'a pas su le résoudre ». Sans cette
		// distinction, un prof sans email suffisait à désaffecter toutes ses
		// séances, et un GRCLE inconnu à élargir l'effectif attendu de la
		// séance à la promotion entière.
		grcleVide := e.Grcle == "" || e.Grcle == "0"
		prcleVide := e.Prcle == ""

		seanceID, err := q.GetSeanceBySource(ctx, GetSeanceBySourceParams{Source: source.Map, ExternalID: plcle})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return res, fmt.Errorf("seance_map: lookup PL=%s: %w", plcle, err)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			seanceID, err = q.CreateSeance(ctx, CreateSeanceParams{
				MatiereID:   matiereID,
				StartsAt:    startsAtPg,
				EndsAt:      endsAt,
				Salle:       salle,
				SalleID:     salleID,
				Prof:        prof,
				PromotionID: promoID,
				GroupeID:    groupeID,
				ProfID:      profID,
				Remarque:    remarque,
			})
			if err != nil {
				return res, err
			}
			// Une séance neuve est toujours à rattacher : une excuse saisie
			// avant sa création couvrirait sinon une plage où elle n'existait
			// pas encore.
			res.aRattacher = append(res.aRattacher, seanceID)
		} else {
			row, err := q.UpdateSeance(ctx, UpdateSeanceParams{
				SeanceID:    seanceID,
				MatiereID:   matiereID,
				StartsAt:    startsAtPg,
				EndsAt:      endsAt,
				Salle:       salle,
				SalleID:     salleID,
				Prof:        prof,
				PromotionID: promoID,
				GroupeID:    groupeID,
				GrcleVide:   grcleVide,
				ProfID:      profID,
				PrcleVide:   prcleVide,
				Remarque:    remarque,
			})
			if err != nil {
				return res, err
			}
			if row.Ressuscitee {
				res.ressuscitees++
			}
			if row.Rattacher {
				res.aRattacher = append(res.aRattacher, seanceID)
			}
		}
		if err = q.UpsertSeanceMap(ctx, UpsertSeanceMapParams{
			InternalID: seanceID,
			Source:     source.Map,
			ExternalID: plcle,
		}); err != nil {
			return res, err
		}
		res.count++
	}
	return res, nil
}

// firstSeanceByCocle parcourt les créneaux collectés et retourne, par COCLE
// (matière amont), l'horaire de début de la première séance rencontrée.
// Les créneaux dont la date/heure est illisible sont ignorés.
func firstSeanceByCocle(creneaux map[string]source.Creneau) map[string]time.Time {
	first := make(map[string]time.Time)
	for _, e := range creneaux {
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
