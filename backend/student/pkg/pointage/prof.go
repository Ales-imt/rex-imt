package pointage

// prof.go — routes Pointage côté prof/gestionnaire (mobile).
//
// Les handlers restent propres à ce service (parsing, autorisation, format de
// réponse) mais toute la donnée et la logique partagées viennent de common :
// requêtes sqlc de back-rex-common/pkg/presencedata, token/code de
// back-rex-common/pkg/presencetoken.
//
// Contrôle d'accès : RequireRoles(PROF, GESTIONNAIRE) au niveau des routes,
// puis appartenance par séance — un prof ne voit/pilote que ses séances
// (seance.prof_id = user id du JWT), un gestionnaire les voit toutes. Le
// filtrage des « séances du jour » se fait côté SQL (variante ByProf).

import (
	"back-rex-common/pkg/auth"
	presencedata "back-rex-common/pkg/presencedata/gen"
	"back-rex-common/pkg/presencetoken"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ─── Response types (alignés sur les DTO du web-admin) ──────────────────────

type seanceJourItem struct {
	ID        int64   `json:"id"`
	MatiereID int64   `json:"matiere_id"`
	Matiere   string  `json:"matiere"`
	StartsAt  string  `json:"starts_at"`
	EndsAt    string  `json:"ends_at"`
	Salle     string  `json:"salle"`
	Prof      string  `json:"prof"`
	Promo     string  `json:"promo"`
	Activated bool    `json:"activated"`
	ClosedAt  *string `json:"closed_at"`
}

type openSeanceResponse struct {
	ID               int64   `json:"id"`
	Code             string  `json:"code"`
	OpenedAt         string  `json:"opened_at"`
	ClosedAt         *string `json:"closed_at"`
	LateAfterMinutes int32   `json:"late_after_minutes"`
}

type tokenResponse struct {
	Token      string `json:"token"`
	Code       string `json:"code"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type elevePresence struct {
	UserID     int32   `json:"user_id"`
	Name       string  `json:"name"`
	Surname    string  `json:"surname"`
	Statut     string  `json:"statut"`
	PointeAt   *string `json:"pointe_at"`
	HorsGroupe bool    `json:"hors_groupe,omitempty"`
}

type presenceResponse struct {
	Matiere  string          `json:"matiere"`
	Total    int             `json:"total"`
	Presents int             `json:"presents"`
	Retards  int             `json:"retards"`
	Absents  int             `json:"absents"`
	Eleves   []elevePresence `json:"eleves"`
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func fmtTs(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func fmtTsPtr(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

func parseSeanceID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "seanceId"), 10, 64)
	return id, err == nil
}

// dayWindow retourne [minuit, minuit+24h) du jour courant en heure de Paris
// (fuseau des plannings, cf. migration/planning.go), quel que soit le fuseau
// du serveur.
func dayWindow() (time.Time, time.Time) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 1)
}

// canManageSeance décide si l'utilisateur peut consulter/piloter une séance :
// gestionnaire → toujours ; prof → uniquement si la séance lui est rattachée
// (prof_id non nul et égal à son user id). Fonction pure, testée unitairement.
func canManageSeance(isGestionnaire bool, userID int, profID pgtype.Int4) bool {
	if isGestionnaire {
		return true
	}
	return profID.Valid && profID.Int32 == int32(userID)
}

// loadManagedSeance charge la séance et applique canManageSeance.
// Écrit la réponse d'erreur (404/403/500) et retourne ok=false le cas échéant.
func loadManagedSeance(w http.ResponseWriter, r *http.Request, seanceID int64) (presencedata.GetSeanceRow, bool) {
	q := presencedata.New(services.GetPgCtx(r.Context()).Db)
	seance, err := q.GetSeance(context.Background(), seanceID)
	if errors.Is(err, pgx.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		return presencedata.GetSeanceRow{}, false
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return presencedata.GetSeanceRow{}, false
	}

	userID := auth.GetSecurityUserId(r.Context())
	if !canManageSeance(auth.HasRole(r.Context(), auth.RoleGestionnaire), *userID, seance.ProfID) {
		services.AuthorizationError(w, r, "séance d'un autre enseignant", services.NO_INFORMATION, nil)
		return presencedata.GetSeanceRow{}, false
	}
	return seance, true
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// GetSeancesJourHandler liste les cours du jour : toutes les séances pour un
// gestionnaire, uniquement les siennes pour un prof (filtrage en SQL).
// GET /pointage/seances/jour
func GetSeancesJourHandler(w http.ResponseWriter, r *http.Request) {
	q := presencedata.New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()
	dayStart, dayEnd := dayWindow()

	tsStart := pgtype.Timestamptz{Time: dayStart, Valid: true}
	tsEnd := pgtype.Timestamptz{Time: dayEnd, Valid: true}

	items := make([]seanceJourItem, 0)
	if auth.HasRole(r.Context(), auth.RoleGestionnaire) {
		rows, err := q.ListSeancesJour(ctx, presencedata.ListSeancesJourParams{
			DayStart: tsStart,
			DayEnd:   tsEnd,
		})
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		for _, row := range rows {
			items = append(items, seanceJourItem{
				ID: row.ID, MatiereID: row.MatiereID, Matiere: row.MatiereName,
				StartsAt: fmtTs(row.StartsAt), EndsAt: fmtTs(row.EndsAt),
				Salle: row.Salle, Prof: row.Prof, Promo: row.Promo,
				Activated: row.Activated, ClosedAt: fmtTsPtr(row.ClosedAt),
			})
		}
	} else {
		userID := auth.GetSecurityUserId(r.Context())
		rows, err := q.ListSeancesJourByProf(ctx, presencedata.ListSeancesJourByProfParams{
			ProfID:   pgtype.Int4{Int32: int32(*userID), Valid: true},
			DayStart: tsStart,
			DayEnd:   tsEnd,
		})
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		for _, row := range rows {
			items = append(items, seanceJourItem{
				ID: row.ID, MatiereID: row.MatiereID, Matiere: row.MatiereName,
				StartsAt: fmtTs(row.StartsAt), EndsAt: fmtTs(row.EndsAt),
				Salle: row.Salle, Prof: row.Prof, Promo: row.Promo,
				Activated: row.Activated, ClosedAt: fmtTsPtr(row.ClosedAt),
			})
		}
	}
	render.JSON(w, r, items)
}

// GetSeancePresenceHandler retourne l'état de présence d'une séance, au même
// format que l'endpoint admin (compteurs + élèves, hors-groupe inclus).
// GET /pointage/seance/{seanceId}/presence
func GetSeancePresenceHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}
	seance, ok := loadManagedSeance(w, r, seanceID)
	if !ok {
		return
	}

	q := presencedata.New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	rows, err := q.ListPresence(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	var presents, retards, absents int
	eleves := make([]elevePresence, 0, len(rows))
	for _, row := range rows {
		switch row.Statut {
		case "PRESENT":
			presents++
		case "RETARD":
			retards++
		default:
			absents++
		}
		eleves = append(eleves, elevePresence{
			UserID:   row.UserID,
			Name:     row.Name,
			Surname:  row.Surname,
			Statut:   row.Statut,
			PointeAt: fmtTsPtr(row.PointeAt),
		})
	}

	horsGroupe, err := q.ListPresenceHorsGroupe(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	for _, row := range horsGroupe {
		eleves = append(eleves, elevePresence{
			UserID:     row.UserID,
			Name:       row.Name,
			Surname:    row.Surname,
			Statut:     row.Statut,
			PointeAt:   fmtTsPtr(row.PointeAt),
			HorsGroupe: true,
		})
	}

	render.JSON(w, r, presenceResponse{
		Matiere:  seance.MatiereName,
		Total:    len(rows),
		Presents: presents,
		Retards:  retards,
		Absents:  absents,
		Eleves:   eleves,
	})
}

type openSeanceRequest struct {
	LateAfterMinutes int32 `json:"late_after_minutes"`
}

// OpenSeanceProfHandler ouvre le pointage d'une séance déjà planifiée (les
// séances du mobile viennent du planning et ont un id — pas de création par
// créneau comme côté admin). Idempotent : si la séance est déjà ouverte, son
// état courant est retourné.
// POST /pointage/seance/{seanceId}/open
func OpenSeanceProfHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}

	var req openSeanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		services.InvalidRequestError(w, r, "corps invalide", services.NO_INFORMATION, nil)
		return
	}
	if req.LateAfterMinutes == 0 {
		req.LateAfterMinutes = 10
	}

	seance, ok := loadManagedSeance(w, r, seanceID)
	if !ok {
		return
	}
	if seance.ClosedAt.Valid {
		services.InvalidRequestError(w, r, "séance déjà clôturée", services.NO_INFORMATION, nil)
		return
	}
	if seance.Code.Valid {
		// Déjà ouverte (idempotent)
		render.JSON(w, r, openSeanceResponse{
			ID:               seance.ID,
			Code:             seance.Code.String,
			OpenedAt:         fmtTs(seance.OpenedAt),
			LateAfterMinutes: seance.LateAfterMinutes,
		})
		return
	}

	q := presencedata.New(services.GetPgCtx(r.Context()).Db)
	activated, err := q.ActivateSeance(context.Background(), presencedata.ActivateSeanceParams{
		ID:               seanceID,
		Code:             pgtype.Text{String: presencetoken.GenerateCode(), Valid: true},
		LateAfterMinutes: req.LateAfterMinutes,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Activée entre-temps par un autre appel : renvoyer l'état courant.
		if current, ok2 := loadManagedSeance(w, r, seanceID); ok2 {
			render.JSON(w, r, openSeanceResponse{
				ID:               current.ID,
				Code:             current.Code.String,
				OpenedAt:         fmtTs(current.OpenedAt),
				LateAfterMinutes: current.LateAfterMinutes,
			})
		}
		return
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, openSeanceResponse{
		ID:               activated.ID,
		Code:             activated.Code.String,
		OpenedAt:         fmtTs(activated.OpenedAt),
		LateAfterMinutes: req.LateAfterMinutes,
	})
}

// CloseSeanceProfHandler clôture une séance (idempotent, comme côté admin).
// POST /pointage/seance/{seanceId}/close
func CloseSeanceProfHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}
	if _, ok := loadManagedSeance(w, r, seanceID); !ok {
		return
	}

	q := presencedata.New(services.GetPgCtx(r.Context()).Db)
	if err := q.CloseSeance(context.Background(), seanceID); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetSeanceTokenHandler émet le token QR tournant + le code de repli, au même
// format que l'endpoint admin.
// GET /pointage/seance/{seanceId}/token
func GetSeanceTokenHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}
	seance, ok := loadManagedSeance(w, r, seanceID)
	if !ok {
		return
	}
	if !seance.Code.Valid {
		services.InvalidRequestError(w, r, "séance non ouverte", services.NO_INFORMATION, nil)
		return
	}
	if seance.ClosedAt.Valid {
		services.InvalidRequestError(w, r, "séance fermée", services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, tokenResponse{
		Token:      presencetoken.Issue(seanceID),
		Code:       seance.Code.String,
		TTLSeconds: int(presencetoken.TTL.Seconds()),
	})
}
