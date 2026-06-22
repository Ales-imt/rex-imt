package presence

import (
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgtype"
)

// ─── Code court ──────────────────────────────────────────────────────────────

const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func generateCode() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = codeAlphabet[rand.Intn(len(codeAlphabet))]
	}
	return string(b)
}

// ─── Response types ───────────────────────────────────────────────────────────

type openSeanceRequest struct {
	MatiereID        int64  `json:"matiere_id"`
	StartsAt         string `json:"starts_at"`
	EndsAt           string `json:"ends_at"`
	Salle            string `json:"salle"`
	Prof             string `json:"prof"`
	LateAfterMinutes int32  `json:"late_after_minutes"`
}

type openSeanceResponse struct {
	ID               int64  `json:"id"`
	Code             string `json:"code"`
	OpenedAt         string `json:"opened_at"`
	LateAfterMinutes int32  `json:"late_after_minutes"`
}

type tokenResponse struct {
	Token      string `json:"token"`
	Code       string `json:"code"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type elevePresence struct {
	UserID   int32   `json:"user_id"`
	Name     string  `json:"name"`
	Surname  string  `json:"surname"`
	Statut   string  `json:"statut"`
	PointeAt *string `json:"pointe_at"`
}

type presenceResponse struct {
	Matiere  string          `json:"matiere"`
	Total    int             `json:"total"`
	Presents int             `json:"presents"`
	Retards  int             `json:"retards"`
	Absents  int             `json:"absents"`
	Eleves   []elevePresence `json:"eleves"`
}

type seanceListItem struct {
	ID       int64   `json:"id"`
	Code     string  `json:"code"`
	OpenedAt string  `json:"opened_at"`
	ClosedAt *string `json:"closed_at"`
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

// academicYear retourne start/end (YYYYMMDD) de l'année scolaire en cours.
// Règle : l'année scolaire commence le 1er septembre.
func academicYear() (start, end string) {
	now := time.Now()
	year := now.Year()
	if now.Month() < time.September {
		year--
	}
	return fmt.Sprintf("%d0901", year), fmt.Sprintf("%d0831", year+1)
}

func parseSeanceID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "seanceId"), 10, 64)
	return id, err == nil
}

func parseISO(s string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func toTstz(s string) pgtype.Timestamptz {
	if s == "" {
		return pgtype.Timestamptz{}
	}
	t, ok := parseISO(s)
	if !ok {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

func GetPlanningHandler(planningURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		matiereID, err := strconv.ParseInt(chi.URLParam(r, "matiereId"), 10, 64)
		if err != nil {
			services.InvalidRequestError(w, r, "matiereId invalide", services.NO_INFORMATION, nil)
			return
		}

		q := New(services.GetPgCtx(r.Context()).Db)
		externalID, err := q.GetMatiereExternalID(context.Background(), matiereID)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		if start == "" || end == "" {
			start, end = academicYear()
		}

		seances, err := FetchSeancesMatiere(r.Context(), planningURL, externalID, start, end)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		render.JSON(w, r, seances)
	}
}

func OpenSeanceHandler(w http.ResponseWriter, r *http.Request) {
	var req openSeanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		services.InvalidRequestError(w, r, "corps invalide", services.NO_INFORMATION, nil)
		return
	}
	if req.MatiereID == 0 {
		services.InvalidRequestError(w, r, "matiere_id requis", services.NO_INFORMATION, nil)
		return
	}
	if req.LateAfterMinutes == 0 {
		req.LateAfterMinutes = 10
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	row, err := q.OpenSeance(context.Background(), OpenSeanceParams{
		MatiereID:        req.MatiereID,
		Code:             generateCode(),
		StartsAt:         toTstz(req.StartsAt),
		EndsAt:           toTstz(req.EndsAt),
		Salle:            toText(req.Salle),
		Prof:             toText(req.Prof),
		LateAfterMinutes: req.LateAfterMinutes,
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, openSeanceResponse{
		ID:               row.ID,
		Code:             row.Code,
		OpenedAt:         fmtTs(row.OpenedAt),
		LateAfterMinutes: req.LateAfterMinutes,
	})
}

func CloseSeanceHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	if err := q.CloseSeance(context.Background(), seanceID); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func GetTokenHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	seance, err := q.GetSeance(context.Background(), seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if seance.ClosedAt.Valid {
		services.InvalidRequestError(w, r, "séance fermée", services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, tokenResponse{
		Token:      IssueToken(seanceID),
		Code:       seance.Code,
		TTLSeconds: int(tokenTTL.Seconds()),
	})
}

func GetPresenceHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	seance, err := q.GetSeance(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	rows, err := q.ListPresence(ctx, ListPresenceParams{
		SeanceID:  seanceID,
		MatiereID: seance.MatiereID,
	})
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

	render.JSON(w, r, presenceResponse{
		Matiere:  seance.MatiereName,
		Total:    len(rows),
		Presents: presents,
		Retards:  retards,
		Absents:  absents,
		Eleves:   eleves,
	})
}

func ListSeancesHandler(w http.ResponseWriter, r *http.Request) {
	matiereID, err := strconv.ParseInt(chi.URLParam(r, "matiereId"), 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "matiereId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	rows, err := q.ListSeancesByMatiere(context.Background(), matiereID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	result := make([]seanceListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, seanceListItem{
			ID:       row.ID,
			Code:     row.Code,
			OpenedAt: fmtTs(row.OpenedAt),
			ClosedAt: fmtTsPtr(row.ClosedAt),
		})
	}
	render.JSON(w, r, result)
}
