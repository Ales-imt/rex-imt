package planning

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

// ─── Types de réponse (format attendu par le front def.ts) ──────────────────────

type horaire struct {
	Lower string `json:"Lower"`
	Upper string `json:"Upper"`
}

type salleRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type intervenantRef struct {
	ID        int32  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type groupeRef struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	OptionID int64  `json:"option_id"`
}

type reservationDetail struct {
	ID           int64            `json:"id"`
	Version      int              `json:"version"`
	Horaire      horaire          `json:"horaire"`
	PeriodeID    int64            `json:"periode_id"`
	MatiereID    int64            `json:"matiere_id"`
	MatiereName  string           `json:"matiere_name"`
	MatiereColor *string          `json:"matiere_color"`
	TypeCours    *string          `json:"type_cours"`
	IsDistanciel bool             `json:"is_distanciel"`
	Description  *string          `json:"description"`
	Salles       []salleRef       `json:"salles"`
	Intervenants []intervenantRef `json:"intervenants"`
	Groupes      []groupeRef      `json:"groupes"`
	Remarque     *string          `json:"remarque"`
}

// heuresItem : une entrée de répartition (par matière, groupe ou prof).
// Pour la dimension « groupe », Children détaille les heures par matière.
type heuresItem struct {
	ID               int64        `json:"id"`
	Label            string       `json:"label"`
	HeuresConsommees float64      `json:"heures_consommees"`
	Children         []heuresItem `json:"children,omitempty"`
}

// heuresBreakdown : les trois partitions d'un même total (matière / groupe / prof).
type heuresBreakdown struct {
	Matiere []heuresItem `json:"matiere"`
	Groupe  []heuresItem `json:"groupe"`
	Prof    []heuresItem `json:"prof"`
}

// occupationSalle : le bilan d'une salle sur la plage demandée. Le taux
// d'occupation n'est PAS calculé ici : il dépend d'une amplitude d'ouverture
// (8h-18h × 5 jours…) qui est une convention d'établissement, paramétrable
// côté front sans redéploiement.
type occupationSalle struct {
	SalleID   int64   `json:"salle_id"`
	SalleName string  `json:"salle_name"`
	Capacite  *int32  `json:"capacite"`
	Type      *string `json:"type"`
	NbSeances int64   `json:"nb_seances"`
	Heures    float64 `json:"heures"`
}

// creneauSalle : un créneau réservé, pour la vue calendrier.
type creneauSalle struct {
	SalleID       int64   `json:"salle_id"`
	SalleName     string  `json:"salle_name"`
	SeanceID      int64   `json:"seance_id"`
	StartsAt      string  `json:"starts_at"`
	EndsAt        string  `json:"ends_at"`
	MatiereName   string  `json:"matiere_name"`
	Prof          *string `json:"prof"`
	GroupeName    *string `json:"groupe_name"`
	PromotionName *string `json:"promotion_name"`
}

// ─── Helpers ────────────────────────────────────────────────────────────────────

// parsePeriodeID lit et valide le paramètre ?periode_id=.
func parsePeriodeID(r *http.Request) (pgtype.Int8, bool) {
	raw := r.URL.Query().Get("periode_id")
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return pgtype.Int8{}, false
	}
	return pgtype.Int8{Int64: v, Valid: true}, true
}

func isoOrEmpty(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Format(time.RFC3339)
}

// parisLoc : les bornes en date seule (« 2026-09-01 ») s'entendent à minuit
// heure de l'établissement, pas UTC — le fallback UTC ne décale les bornes que
// si la timezone est indisponible sur la machine.
var parisLoc = func() *time.Location {
	l, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.UTC
	}
	return l
}()

// parsePlage lit les paramètres ?debut=&fin= — RFC 3339, ou date seule
// AAAA-MM-JJ interprétée à minuit Europe/Paris. La plage est [debut, fin[ :
// pour une semaine, fin est le lundi suivant.
func parsePlage(r *http.Request) (debut, fin pgtype.Timestamptz, ok bool) {
	parse := func(raw string) (pgtype.Timestamptz, bool) {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return pgtype.Timestamptz{Time: t, Valid: true}, true
		}
		if t, err := time.ParseInLocation("2006-01-02", raw, parisLoc); err == nil {
			return pgtype.Timestamptz{Time: t, Valid: true}, true
		}
		return pgtype.Timestamptz{}, false
	}
	debut, okD := parse(r.URL.Query().Get("debut"))
	fin, okF := parse(r.URL.Query().Get("fin"))
	return debut, fin, okD && okF && debut.Time.Before(fin.Time)
}

func textOrNil(t pgtype.Text) *string {
	if !t.Valid || t.String == "" {
		return nil
	}
	return &t.String
}

// ─── Handlers ───────────────────────────────────────────────────────────────────

func GetReservations(w http.ResponseWriter, r *http.Request) {
	periodeID, ok := parsePeriodeID(r)
	if !ok {
		services.InvalidRequestError(w, r, "periode_id invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	rows, err := q.GetReservationsByPeriode(context.Background(), periodeID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	result := make([]reservationDetail, 0, len(rows))
	for _, row := range rows {
		res := reservationDetail{
			ID:           row.ID,
			Version:      0,
			Horaire:      horaire{Lower: isoOrEmpty(row.StartsAt), Upper: isoOrEmpty(row.EndsAt)},
			PeriodeID:    periodeID.Int64,
			MatiereID:    row.MatiereID,
			MatiereName:  row.MatiereName,
			MatiereColor: nil,
			TypeCours:    nil,
			IsDistanciel: false,
			Description:  nil,
			Salles:       []salleRef{},
			Intervenants: []intervenantRef{},
			Groupes:      []groupeRef{},
		}

		if row.SalleID.Valid {
			res.Salles = append(res.Salles, salleRef{ID: row.SalleID.Int64, Name: row.SalleName.String})
		}
		if row.Prof.Valid && row.Prof.String != "" {
			var profID int32
			if row.ProfID.Valid {
				profID = row.ProfID.Int32
			}
			res.Intervenants = append(res.Intervenants, intervenantRef{ID: profID, FirstName: row.Prof.String, LastName: ""})
		}
		if row.GroupeID.Valid {
			name := ""
			if row.GroupeName.Valid {
				name = row.GroupeName.String
			}
			res.Groupes = append(res.Groupes, groupeRef{ID: row.GroupeID.Int64, Name: name, OptionID: 0})
		}
		if row.Remarque.Valid && row.Remarque.String != "" {
			res.Remarque = &row.Remarque.String
		}

		result = append(result, res)
	}

	render.JSON(w, r, result)
}

// GetOccupation rend les heures réservées par salle sur [debut, fin[. La
// requête part des SALLES : une salle jamais réservée sort à 0h, et c'est
// précisément l'information qu'une liste construite depuis seance ne pouvait
// pas donner.
func GetOccupation(w http.ResponseWriter, r *http.Request) {
	debut, fin, ok := parsePlage(r)
	if !ok {
		services.InvalidRequestError(w, r, "debut/fin invalides (RFC 3339 ou AAAA-MM-JJ, debut < fin)", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	rows, err := q.GetOccupationSalles(context.Background(), GetOccupationSallesParams{Debut: debut, Fin: fin})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	result := make([]occupationSalle, 0, len(rows))
	for _, row := range rows {
		o := occupationSalle{
			SalleID:   row.SalleID,
			SalleName: row.SalleName,
			Type:      textOrNil(row.Type),
			NbSeances: row.NbSeances,
			Heures:    row.Heures,
		}
		if row.Capacite.Valid {
			o.Capacite = &row.Capacite.Int32
		}
		result = append(result, o)
	}
	render.JSON(w, r, result)
}

// GetCreneaux rend le détail des créneaux par salle, pour la vue calendrier.
func GetCreneaux(w http.ResponseWriter, r *http.Request) {
	debut, fin, ok := parsePlage(r)
	if !ok {
		services.InvalidRequestError(w, r, "debut/fin invalides (RFC 3339 ou AAAA-MM-JJ, debut < fin)", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	rows, err := q.GetCreneauxSalles(context.Background(), GetCreneauxSallesParams{Debut: debut, Fin: fin})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	result := make([]creneauSalle, 0, len(rows))
	for _, row := range rows {
		result = append(result, creneauSalle{
			SalleID:       row.SalleID,
			SalleName:     row.SalleName,
			SeanceID:      row.SeanceID,
			StartsAt:      isoOrEmpty(row.StartsAt),
			EndsAt:        isoOrEmpty(row.EndsAt),
			MatiereName:   row.MatiereName,
			Prof:          textOrNil(row.Prof),
			GroupeName:    textOrNil(row.GroupeName),
			PromotionName: textOrNil(row.PromotionName),
		})
	}
	render.JSON(w, r, result)
}

func GetHeures(w http.ResponseWriter, r *http.Request) {
	periodeID, ok := parsePeriodeID(r)
	if !ok {
		services.InvalidRequestError(w, r, "periode_id invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	matiereRows, err := q.GetHeuresConsommeesByPeriode(ctx, periodeID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	groupeRows, err := q.GetHeuresConsommeesByGroupe(ctx, periodeID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	profRows, err := q.GetHeuresConsommeesByProf(ctx, periodeID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	result := heuresBreakdown{
		Matiere: make([]heuresItem, 0, len(matiereRows)),
		Groupe:  make([]heuresItem, 0, len(groupeRows)),
		Prof:    make([]heuresItem, 0, len(profRows)),
	}

	// Matière : nichée par groupe. Les lignes arrivent triées par matière puis
	// groupe ; on agrège en conservant cet ordre.
	matiereIdx := map[int64]int{}
	for _, row := range matiereRows {
		mi, ok := matiereIdx[row.MatiereID]
		if !ok {
			result.Matiere = append(result.Matiere, heuresItem{ID: row.MatiereID, Label: row.MatiereName, Children: []heuresItem{}})
			mi = len(result.Matiere) - 1
			matiereIdx[row.MatiereID] = mi
		}
		gid := int64(0)
		glabel := "Sans groupe"
		if row.GroupeID.Valid {
			gid = row.GroupeID.Int64
			if row.GroupeName.Valid && row.GroupeName.String != "" {
				glabel = row.GroupeName.String
			}
		}
		result.Matiere[mi].HeuresConsommees += row.HeuresConsommees
		result.Matiere[mi].Children = append(result.Matiere[mi].Children, heuresItem{
			ID:               gid,
			Label:            glabel,
			HeuresConsommees: row.HeuresConsommees,
		})
	}
	// Groupe : niché par matière. Les lignes arrivent triées par groupe puis
	// matière ; on agrège en conservant cet ordre.
	groupeIdx := map[int64]int{}
	for _, row := range groupeRows {
		id := int64(0)
		label := "Sans groupe"
		if row.GroupeID.Valid {
			id = row.GroupeID.Int64
			if row.GroupeName.Valid && row.GroupeName.String != "" {
				label = row.GroupeName.String
			}
		}
		gi, ok := groupeIdx[id]
		if !ok {
			result.Groupe = append(result.Groupe, heuresItem{ID: id, Label: label, Children: []heuresItem{}})
			gi = len(result.Groupe) - 1
			groupeIdx[id] = gi
		}
		result.Groupe[gi].HeuresConsommees += row.HeuresConsommees
		result.Groupe[gi].Children = append(result.Groupe[gi].Children, heuresItem{
			ID:               row.MatiereID,
			Label:            row.MatiereName,
			HeuresConsommees: row.HeuresConsommees,
		})
	}
	for _, row := range profRows {
		id := int64(0)
		if row.ProfID.Valid {
			id = int64(row.ProfID.Int32)
		}
		label := "Sans intervenant"
		if row.Prof.Valid && row.Prof.String != "" {
			label = row.Prof.String
		}
		result.Prof = append(result.Prof, heuresItem{
			ID:               id,
			Label:            label,
			HeuresConsommees: row.HeuresConsommees,
		})
	}

	render.JSON(w, r, result)
}
