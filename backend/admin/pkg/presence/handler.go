package presence

import (
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
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

type seancePlanningItem struct {
	ID       int64  `json:"id"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
	Salle    string `json:"salle"`
	Prof     string `json:"prof"`
	Promo    string `json:"promo"`
}

func GetPlanningHandler(w http.ResponseWriter, r *http.Request) {
	matiereID, err := strconv.ParseInt(chi.URLParam(r, "matiereId"), 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "matiereId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	rows, err := q.ListSeancesPlanning(context.Background(), matiereID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	result := make([]seancePlanningItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, seancePlanningItem{
			ID:       row.ID,
			StartsAt: fmtTs(row.StartsAt),
			EndsAt:   fmtTs(row.EndsAt),
			Salle:    row.Salle,
			Prof:     row.Prof,
			Promo:    row.Promo,
		})
	}
	render.JSON(w, r, result)
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
	ctx := context.Background()

	startsAt := toTstz(req.StartsAt)
	if startsAt.Valid {
		existing, err := q.GetSeanceBySlot(ctx, GetSeanceBySlotParams{
			MatiereID: req.MatiereID,
			StartsAt:  startsAt,
		})
		if err == nil {
			if existing.ClosedAt.Valid {
				services.InvalidRequestError(w, r, "ce créneau a déjà été clôturé", services.NO_INFORMATION, nil)
				return
			}
			if existing.Code.Valid {
				// Séance déjà ouverte (idempotent)
				render.JSON(w, r, openSeanceResponse{
					ID:               existing.ID,
					Code:             existing.Code.String,
					OpenedAt:         fmtTs(existing.OpenedAt),
					LateAfterMinutes: existing.LateAfterMinutes,
				})
				return
			}
			// Séance importée, pas encore activée → on lui assigne un code
			activated, err := q.ActivateSeance(ctx, ActivateSeanceParams{
				ID:               existing.ID,
				Code:             toText(generateCode()),
				LateAfterMinutes: req.LateAfterMinutes,
			})
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
			return
		}
	}

	row, err := q.OpenSeance(ctx, OpenSeanceParams{
		MatiereID:        req.MatiereID,
		Code:             toText(generateCode()),
		StartsAt:         startsAt,
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
		Code:             row.Code.String,
		OpenedAt:         fmtTs(row.OpenedAt),
		LateAfterMinutes: req.LateAfterMinutes,
	})
}

func GetSlotHandler(w http.ResponseWriter, r *http.Request) {
	matiereID, err := strconv.ParseInt(chi.URLParam(r, "matiereId"), 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "matiereId invalide", services.NO_INFORMATION, nil)
		return
	}
	startsAt := toTstz(r.URL.Query().Get("starts_at"))
	if !startsAt.Valid {
		services.InvalidRequestError(w, r, "starts_at invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	row, err := q.GetSeanceBySlot(context.Background(), GetSeanceBySlotParams{
		MatiereID: matiereID,
		StartsAt:  startsAt,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if !row.Code.Valid {
		// Séance importée pas encore activée → traiter comme inexistante
		w.WriteHeader(http.StatusNotFound)
		return
	}

	render.JSON(w, r, openSeanceResponse{
		ID:               row.ID,
		Code:             row.Code.String,
		OpenedAt:         fmtTs(row.OpenedAt),
		ClosedAt:         fmtTsPtr(row.ClosedAt),
		LateAfterMinutes: row.LateAfterMinutes,
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
		Code:       seance.Code.String,
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
			Code:     row.Code.String,
			OpenedAt: fmtTs(row.OpenedAt),
			ClosedAt: fmtTsPtr(row.ClosedAt),
		})
	}
	render.JSON(w, r, result)
}

// ─── Registre d'intégrité ─────────────────────────────────────────────────────

type verifyResponse struct {
	Chain   VerifyChainResult    `json:"chain"`
	Anchors []AnchorVerifyResult `json:"anchors"`
}

// LedgerVerifyHandler vérifie l'intégrité de la chaîne et des ancres TSA.
// GET /presence/ledger/verify
func LedgerVerifyHandler(cfg timestampCfg) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := services.GetPgCtx(r.Context()).Db
		ctx := context.Background()

		chainResult, err := VerifyChain(ctx, db)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		anchorResults, err := VerifyAnchors(ctx, db, cfg.caCertPath)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		render.JSON(w, r, verifyResponse{
			Chain:   chainResult,
			Anchors: anchorResults,
		})
	}
}

type anchorResponse struct {
	Results []AnchorResult `json:"results"`
}

// LedgerAnchorHandler déclenche un ancrage TSA manuel sur le dernier maillon.
// POST /presence/ledger/anchor
func LedgerAnchorHandler(cfg timestampCfg) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := services.GetPgCtx(r.Context()).Db
		ctx := context.Background()

		results, err := AnchorLast(ctx, db, cfg.tsCfg)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		render.JSON(w, r, anchorResponse{Results: results})
	}
}

// PresencePdfHandler génère une feuille de présence PDF pour une séance clôturée.
// GET /presence/seance/{seanceId}/pdf
func PresencePdfHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	seance, err := q.GetSeanceDetail(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if !seance.ClosedAt.Valid {
		http.Error(w, "la séance doit être clôturée pour générer le PDF", http.StatusUnprocessableEntity)
		return
	}

	rows, err := q.ListPresence(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	horsGroupe, err := q.ListPresenceHorsGroupe(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	data := buildPdfData(seance, rows, horsGroupe)

	filename := fmt.Sprintf("presence-seance-%d.pdf", seanceID)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-store")

	if err := generatePresencePDF(w, data); err != nil {
		// Headers already sent; log only
		_ = err
	}
}

// timestampCfg regroupe la configuration TSA passée aux handlers.
type timestampCfg struct {
	tsCfg      services.TimestampConfig
	caCertPath string
}
