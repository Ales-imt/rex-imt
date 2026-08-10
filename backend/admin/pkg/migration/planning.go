package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// planningEntry regroupe les données utiles d'une ligne PL du planning.
type planningEntry struct {
	plcle  string // identifiant stable du créneau (champ "PL" de webdfd)
	p0cle  string // promo webdfd
	cocle  string // matière webdfd
	grcle  string // groupe webdfd
	prcle  string // identifiant stable du prof (champ "PRCLE" de webdfd)
	cours  string // nom d'affichage de la matière (tronqué)
	groupe string // nom d'affichage du groupe
	date   string // AAAAMMJJ
	hd     string // HHMM heure de début
	hf     string // HHMM heure de fin
	salle  string // salle
	prof   string // nom d'affichage du professeur
}

// SyncPlanning charge le planning de chaque promo connue (planning_txt),
// puis enrichit les noms via cours_txt, et upserte groupes + matières.
//
// Ordre : 1. Récupérer cours_txt (noms complets). 2. Pour chaque promo,
// parser planning_txt. 3. Déduplication puis upsert en base.
func SyncPlanning(ctx context.Context, baseURL string, db *pgxpool.Pool, ac anneeCourante) error {
	q := New(db)
	annee := ac.Annee

	// --- 1. Noms complets depuis cours_txt ---
	coursNoms, err := fetchCoursNoms(baseURL + "?TYPE=cours_txt")
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

	// seanceKeys : clé "{cocle}_{date}_{hd}" → première occurrence de planningEntry.
	// Correspond à l'unicité (matiere_id, starts_at) dans public.seance.
	seanceKeys := make(map[string]planningEntry)

	startAnnee := ac.Debut.Format("20060102")
	endAnnee := ac.Fin.Format("20060102")

	for _, promo := range promos {
		entries, err := fetchPlanning(baseURL, promo.ExternalID, startAnnee, endAnnee)
		if err != nil {
			log.Printf("planning: promo P0=%s inaccessible: %v — ignorée", promo.ExternalID, err)
			continue
		}
		for _, e := range entries {
			if e.cocle != "" && e.cocle != "0" {
				if _, seen := cocles[e.cocle]; !seen {
					cocles[e.cocle] = cocleInfo{
						displayName: e.cours,
						promoExtID:  e.p0cle,
						promoID:     promo.InternalID,
					}
				}
				if e.plcle != "" {
					seanceKeys[e.plcle] = e
				}
			}
			if e.grcle != "" && e.grcle != "0" {
				if _, seen := grcles[e.grcle]; !seen {
					grcles[e.grcle] = grcleInfo{
						displayName: e.groupe,
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

		matiereID, err := q.GetMatiereBySource(ctx, GetMatiereBySourceParams{Source: "webdfd", ExternalID: cocleStr, Annee: annee})
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
			Source:     "webdfd",
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
		groupeID, err := q.GetGroupeBySource(ctx, GetGroupeBySourceParams{Source: "webdfd", ExternalID: grcleStr, Annee: annee})
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
			Source:     "webdfd",
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
		if e.prcle == "" {
			continue
		}
		if _, already := prcleToUserID[e.prcle]; already {
			continue
		}
		uid, err := q.GetProfBySource(ctx, GetProfBySourceParams{Source: "webdfd", ExternalID: e.prcle})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("prof_map: lookup PRCLE=%s depuis planning: %w", e.prcle, err)
		}
		if err == nil {
			prcleToUserID[e.prcle] = uid
		}
	}

	// --- 7. Upsert séances ---
	sCount, sErr := syncSeances(ctx, q, seanceKeys, cocleToMatiereID, grcleToGroupeID, prcleToUserID, promos)
	log.Printf("planning: %d séances synchronisées (annee=%d)", sCount, annee)
	return sErr
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

func parseWebdfdTime(date, hhmm string) (time.Time, bool) {
	if len(date) != 8 || len(hhmm) != 4 {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(planningTimeFmt, date+" "+hhmm, parisLoc)
	return t, err == nil
}

// syncSeances synchronise chaque créneau webdfd (identifié par PL = plcle stable)
// dans public.seance + migration.seance_map.
// Schéma : lookup seance_map par PL → trouvé : UPDATE ; absent : INSERT.
func syncSeances(
	ctx context.Context,
	q *Queries,
	seanceKeys map[string]planningEntry,
	cocleToMatiereID map[string]int64,
	grcleToGroupeID map[string]int64,
	prcleToUserID map[string]int32,
	promos []ListPromotionWebdfdIDsRow,
) (int, error) {
	p0cleToPromoID := make(map[string]int64, len(promos))
	for _, p := range promos {
		p0cleToPromoID[p.ExternalID] = p.InternalID
	}

	var count int
	for plcle, e := range seanceKeys {
		matiereID, ok := cocleToMatiereID[e.cocle]
		if !ok {
			continue
		}
		startsAt, ok := parseWebdfdTime(e.date, e.hd)
		if !ok {
			continue
		}
		var endsAt pgtype.Timestamptz
		if t, ok2 := parseWebdfdTime(e.date, e.hf); ok2 {
			endsAt = pgtype.Timestamptz{Time: t, Valid: true}
		}
		var promoID pgtype.Int8
		if v := p0cleToPromoID[e.p0cle]; v != 0 {
			promoID = pgtype.Int8{Int64: v, Valid: true}
		}
		var groupeID pgtype.Int8
		if v := grcleToGroupeID[e.grcle]; v != 0 {
			groupeID = pgtype.Int8{Int64: v, Valid: true}
		}
		salle := pgtype.Text{String: e.salle, Valid: e.salle != ""}
		prof := pgtype.Text{String: e.prof, Valid: e.prof != ""}
		var profID pgtype.Int4
		if uid, found := prcleToUserID[e.prcle]; found {
			profID = pgtype.Int4{Int32: uid, Valid: true}
		}
		startsAtPg := pgtype.Timestamptz{Time: startsAt, Valid: true}

		seanceID, err := q.GetSeanceBySource(ctx, GetSeanceBySourceParams{Source: "webdfd", ExternalID: plcle})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return count, fmt.Errorf("seance_map: lookup PL=%s: %w", plcle, err)
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
			err = q.UpdateSeance(ctx, UpdateSeanceParams{
				ID:          seanceID,
				MatiereID:   matiereID,
				StartsAt:    startsAtPg,
				EndsAt:      endsAt,
				Salle:       salle,
				Prof:        prof,
				PromotionID: promoID,
				GroupeID:    groupeID,
				ProfID:      profID,
			})
		}
		if err != nil {
			return count, err
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
			return count, fmt.Errorf("justification_seance: séance %d: %w", seanceID, err)
		}
		if err = q.UpsertSeanceMap(ctx, UpsertSeanceMapParams{
			InternalID: seanceID,
			Source:     "webdfd",
			ExternalID: plcle,
		}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// fetchCoursNoms récupère le flux cours_txt et retourne une map COCLE→NOM.
func fetchCoursNoms(url string) (map[string]string, error) {
	resp, err := webdfdGet(url)
	if err != nil {
		return nil, fmt.Errorf("webdfd: cours_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return nil, err
	}

	noms := make(map[string]string)
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		co := strings.TrimSpace(kv["CO"])
		nom := strings.TrimSpace(kv["NOM"])
		if co != "" && nom != "" {
			noms[co] = nom
		}
	}
	return noms, nil
}

// fetchPlanning interroge planning_txt pour une promo donnée sur la plage annuelle.
func fetchPlanning(baseURL, p0cle, debut, fin string) ([]planningEntry, error) {
	url := fmt.Sprintf("%s?TYPE=planning_txt&DATEDEBUT=%s&DATEFIN=%s&TYPECLE=p0cleunik&VALCLE=%s",
		baseURL, debut, fin, p0cle)

	resp, err := webdfdGet(url)
	if err != nil {
		return nil, fmt.Errorf("webdfd: planning_txt P0=%s inaccessible: %w", p0cle, err)
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return nil, err
	}

	var entries []planningEntry
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		if kv["DATE"] == "" {
			continue
		}
		entries = append(entries, planningEntry{
			plcle:  strings.TrimSpace(kv["PL"]),
			p0cle:  strings.TrimSpace(kv["P0CLE"]),
			cocle:  strings.TrimSpace(kv["COCLE"]),
			grcle:  strings.TrimSpace(kv["GRCLE"]),
			prcle:  strings.TrimSpace(kv["PRCLE"]),
			cours:  strings.TrimSpace(kv["COURS"]),
			groupe: strings.TrimSpace(kv["GROUPE"]),
			date:   strings.TrimSpace(kv["DATE"]),
			hd:     strings.TrimSpace(kv["HD"]),
			hf:     strings.TrimSpace(kv["HF"]),
			salle:  strings.TrimSpace(kv["SALLE"]),
			prof:   strings.TrimSpace(kv["PROF"]),
		})
	}
	return entries, nil
}

// firstSeanceByCocle parcourt les créneaux collectés et retourne, par COCLE
// (matière webdfd), l'horaire de début de la première séance rencontrée.
// Les créneaux dont la date/heure est illisible sont ignorés.
func firstSeanceByCocle(seanceKeys map[string]planningEntry) map[string]time.Time {
	first := make(map[string]time.Time)
	for _, e := range seanceKeys {
		if e.cocle == "" || e.cocle == "0" {
			continue
		}
		t, ok := parseWebdfdTime(e.date, e.hd)
		if !ok {
			continue
		}
		if cur, seen := first[e.cocle]; !seen || t.Before(cur) {
			first[e.cocle] = t
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
