package presence

import (
	"back-rex-common/pkg/ledger"
	"back-rex-common/pkg/presencedata"
	presencegen "back-rex-common/pkg/presencedata/gen"
	"back-rex-common/pkg/presencetoken"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

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
	UserID   int32   `json:"user_id"`
	Name     string  `json:"name"`
	Surname  string  `json:"surname"`
	Statut   string  `json:"statut"`
	PointeAt *string `json:"pointe_at"`
	// Justifie : une excuse active couvre cette séance pour cet élève. Ce
	// n'est PAS un statut — le pointage l'emporte toujours, et l'affichage
	// n'en tient compte que lorsque le statut dérivé vaut ABSENT.
	Justifie   bool `json:"justifie"`
	HorsGroupe bool `json:"hors_groupe,omitempty"`
}

// presenceResponse : Absents et Excuses sont disjoints — un absent excusé est
// compté dans Excuses seulement, de sorte que
// Presents + Retards + Absents + Excuses = Total.
type presenceResponse struct {
	Matiere  string          `json:"matiere"`
	Total    int             `json:"total"`
	Presents int             `json:"presents"`
	Retards  int             `json:"retards"`
	Absents  int             `json:"absents"`
	Excuses  int             `json:"excuses"`
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
			activated, err := presencegen.New(services.GetPgCtx(r.Context()).Db).
				ActivateSeance(ctx, presencegen.ActivateSeanceParams{
					ID:               existing.ID,
					Code:             toText(presencetoken.GenerateCode()),
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
		Code:             toText(presencetoken.GenerateCode()),
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

	// Clôture ET figement de l'effectif attendu, dans une seule transaction :
	// la règle vit dans common, jamais dupliquée avec le chemin mobile.
	if _, _, err := presencedata.CloseSeanceEtFiger(
		context.Background(), services.GetPgCtx(r.Context()).Db, seanceID,
	); err != nil {
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

	q := presencegen.New(services.GetPgCtx(r.Context()).Db)
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
		Token:      presencetoken.Issue(seanceID),
		Code:       seance.Code.String,
		TTLSeconds: int(presencetoken.TTL.Seconds()),
	})
}

func GetPresenceHandler(w http.ResponseWriter, r *http.Request) {
	seanceID, ok := parseSeanceID(r)
	if !ok {
		services.InvalidRequestError(w, r, "seanceId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := presencegen.New(services.GetPgCtx(r.Context()).Db)
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

	var presents, retards, absents, excuses int
	eleves := make([]elevePresence, 0, len(rows))
	for _, row := range rows {
		switch {
		case row.Statut == "PRESENT":
			presents++
		case row.Statut == "RETARD":
			retards++
		case row.Justifie:
			excuses++
		default:
			absents++
		}
		eleves = append(eleves, elevePresence{
			UserID:   row.UserID,
			Name:     row.Name,
			Surname:  row.Surname,
			Statut:   row.Statut,
			PointeAt: fmtTsPtr(row.PointeAt),
			Justifie: row.Justifie,
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
			Justifie:   row.Justifie,
			HorsGroupe: true,
		})
	}

	render.JSON(w, r, presenceResponse{
		Matiere:  seance.MatiereName,
		Total:    len(rows),
		Presents: presents,
		Retards:  retards,
		Absents:  absents,
		Excuses:  excuses,
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
	Chain   ledger.VerifyChainResult `json:"chain"`
	Anchors []AnchorVerifyResult     `json:"anchors"`
}

// LedgerVerifyHandler vérifie l'intégrité de la chaîne et des ancres TSA.
// GET /presence/ledger/verify
func LedgerVerifyHandler(cfg timestampCfg) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := services.GetPgCtx(r.Context()).Db
		ctx := context.Background()

		chainResult, err := ledger.VerifyChain(ctx, db)
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

// LedgerAnchorHandler déclenche un ancrage TSA manuel sur le dernier maillon,
// puis l'envoi du témoin externe pour chaque nouvelle ancre. Un échec d'envoi
// ne fait pas échouer l'ancrage (l'ancre en base reste valide).
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

		SendWitnesses(ctx, db, cfg.witness, results)

		render.JSON(w, r, anchorResponse{Results: results})
	}
}

type verifyWitnessRequest struct {
	Token   string `json:"token"`
	TsaCert string `json:"tsa_cert"`
}

// WitnessVerifyHandler confronte un témoin RFC 3161 fourni par l'auditeur
// (jeton reçu par e-mail depuis la boîte externe) à l'état actuel du registre.
// Décodage tolérant du jeton (base64 avec sauts de ligne, PEM) ; le certificat
// TSA est optionnel, à défaut le CA racine caCertPath sert de confiance.
// Lecture seule : rien n'est écrit ni conservé.
// POST /presence/ledger/verify-witness
func WitnessVerifyHandler(cfg timestampCfg) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req verifyWitnessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			services.InvalidRequestError(w, r, "corps invalide", services.NO_INFORMATION, nil)
			return
		}

		tokenDER, err := decodeWitnessToken([]byte(req.Token))
		if err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		tsaCertPEM, err := decodeWitnessCert(req.TsaCert)
		if err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		db := services.GetPgCtx(r.Context()).Db
		result, err := VerifyWitness(context.Background(), db, tokenDER, tsaCertPEM, cfg.caCertPath)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		render.JSON(w, r, result)
	}
}

type anchorRefItem struct {
	LedgerSeq int64  `json:"ledger_seq"`
	CreatedAt string `json:"created_at"`
	TsaUrl    string `json:"tsa_url"`
}

// LedgerAnchorsHandler liste les ancres présentes en base. Pour la page de
// vérification d'un témoin, ce sont de simples repères temporels (intervalle
// entre témoins pour la dichotomie) : ces dates viennent de la base et ne
// constituent pas une preuve — la preuve reste les témoins externes.
// GET /presence/ledger/anchors
func LedgerAnchorsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := New(services.GetPgCtx(r.Context()).Db).ListAnchors(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	items := make([]anchorRefItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, anchorRefItem{
			LedgerSeq: row.LedgerSeq,
			CreatedAt: fmtTs(row.CreatedAt),
			TsaUrl:    row.TsaUrl,
		})
	}
	render.JSON(w, r, items)
}

// WitnessResendHandler retente l'envoi du témoin externe d'une ancre donnée
// (utile si le SMTP était indisponible). Idempotent : les destinataires déjà
// SENT sont ignorés.
// POST /presence/witness/resend/{anchorID}
func WitnessResendHandler(cfg timestampCfg) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		anchorID, err := strconv.ParseInt(chi.URLParam(r, "anchorID"), 10, 64)
		if err != nil {
			services.InvalidRequestError(w, r, "anchorID invalide", services.NO_INFORMATION, nil)
			return
		}
		if !cfg.witness.Enabled {
			services.InvalidRequestError(w, r, "témoin externe désactivé (presence.witness.enabled)", services.NO_INFORMATION, nil)
			return
		}

		db := services.GetPgCtx(r.Context()).Db
		report, err := SendWitnessForAnchor(context.Background(), db, cfg.witness, anchorID)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		render.JSON(w, r, report)
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

	db := services.GetPgCtx(r.Context()).Db
	q := New(db)
	qd := presencegen.New(db)
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

	rows, err := qd.ListPresence(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	horsGroupe, err := qd.ListPresenceHorsGroupe(ctx, seanceID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	lignes := make([]presenceLigne, 0, len(rows))
	for _, r := range rows {
		lignes = append(lignes, presenceLigne{
			UserID: r.UserID, Name: r.Name, Surname: r.Surname,
			Statut: r.Statut, PointeAt: r.PointeAt, Justifie: r.Justifie,
		})
	}
	lignesHG := make([]presenceLigne, 0, len(horsGroupe))
	for _, r := range horsGroupe {
		lignesHG = append(lignesHG, presenceLigne{
			UserID: r.UserID, Name: r.Name, Surname: r.Surname,
			Statut: r.Statut, PointeAt: r.PointeAt, Justifie: r.Justifie,
		})
	}

	data := buildPdfData(seanceInfoFromDetail(seance), lignes, lignesHG, time.Now().UTC())

	filename := fmt.Sprintf("presence-seance-%d.pdf", seanceID)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-store")

	if err := generatePresencePDF(w, data); err != nil {
		// Headers already sent; log only
		_ = err
	}
}

// ─── Export semestriel ────────────────────────────────────────────────────────

// parseBool lit un booléen de query string, avec valeur par défaut lorsque le
// paramètre est absent.
func parseBool(r *http.Request, nom string, defaut bool) bool {
	v := r.URL.Query().Get(nom)
	if v == "" {
		return defaut
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaut
	}
	return b
}

// parseIDs lit une liste d'identifiants en CSV. Une liste vide vaut « toutes
// les matières » : le filtre SQL est alors désactivé.
func parseIDs(csv string) ([]int64, error) {
	if strings.TrimSpace(csv) == "" {
		return nil, nil
	}
	champs := strings.Split(csv, ",")
	ids := make([]int64, 0, len(champs))
	for _, c := range champs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		id, err := strconv.ParseInt(c, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("identifiant de matière invalide : %q", c)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// parseBorneJour accepte une date seule (AAAA-MM-JJ) ou un instant ISO complet.
// Pour la borne haute, une date seule désigne la journée entière : elle est
// avancée au lendemain, la requête comparant par « < ».
func parseBorneJour(s string, finDeJournee bool) (pgtype.Timestamptz, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Timestamptz{}, nil
	}
	if t, ok := parseISO(s); ok {
		return pgtype.Timestamptz{Time: t, Valid: true}, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, parisLoc)
	if err != nil {
		return pgtype.Timestamptz{}, fmt.Errorf("date invalide : %q", s)
	}
	if finDeJournee {
		t = t.AddDate(0, 0, 1)
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}

// PeriodePresencePdfHandler génère en un seul document toutes les feuilles de
// présence d'une période. Génération synchrone en mémoire : de l'ordre de
// 120 à 200 pages en quelques secondes, un traitement asynchrone coûterait
// plus qu'il ne rapporterait.
// GET /presence/periodes/{periodeId}/pdf
func PeriodePresencePdfHandler(w http.ResponseWriter, r *http.Request) {
	periodeID, err := strconv.ParseInt(chi.URLParam(r, "periodeId"), 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "periodeId invalide", services.NO_INFORMATION, nil)
		return
	}

	matiereIDs, err := parseIDs(r.URL.Query().Get("matiere_ids"))
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	from, err := parseBorneJour(r.URL.Query().Get("from"), false)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	to, err := parseBorneJour(r.URL.Query().Get("to"), true)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	doc, header, err := buildSemestreDoc(context.Background(), services.GetPgCtx(r.Context()).Db, exportParams{
		PeriodeID:  periodeID,
		MatiereIDs: matiereIDs,
		From:       from,
		To:         to,
		Options: pdfOptions{
			Cover:     parseBool(r, "cover", true),
			Recap:     parseBool(r, "recap", true),
			Signature: parseBool(r, "signature", false),
			Ledger:    parseBool(r, "ledger", true),
		},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "période introuvable", http.StatusNotFound)
			return
		}
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// Contrairement au PDF unitaire, aucune séance ouverte ne fait échouer la
	// requête : elles sont listées en page de garde (cf. buildSemestreDoc).
	filename := exportFilename(header)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Cache-Control", "no-store")

	if err := generateSemestrePDF(w, doc); err != nil {
		// En-têtes déjà émis : impossible de basculer sur une réponse d'erreur.
		_ = err
	}
}

type exportMatiereItem struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	NbSeances      int64  `json:"nb_seances"`
	NbNonCloturees int64  `json:"nb_non_cloturees"`
	Premiere       string `json:"premiere"`
	Derniere       string `json:"derniere"`
}

type exportSummaryResponse struct {
	PeriodeID int64               `json:"periode_id"`
	Periode   string              `json:"periode"`
	Promo     string              `json:"promo"`
	Annee     int32               `json:"annee"`
	NbEleves  int64               `json:"nb_eleves"`
	Du        string              `json:"du"`
	Au        string              `json:"au"`
	Matieres  []exportMatiereItem `json:"matieres"`
}

// PeriodeExportSummaryHandler renseigne le dialogue d'export avant génération :
// matières, volumétrie, séances encore ouvertes et étendue des dates. Sans lui,
// le client devrait interroger le planning matière par matière.
// GET /presence/periodes/{periodeId}/export-summary
func PeriodeExportSummaryHandler(w http.ResponseWriter, r *http.Request) {
	periodeID, err := strconv.ParseInt(chi.URLParam(r, "periodeId"), 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "periodeId invalide", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	header, err := q.GetPeriodeHeader(ctx, periodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "période introuvable", http.StatusNotFound)
			return
		}
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	matieres, err := q.ListMatieresPeriodeSummary(ctx, pgtype.Int8{Int64: periodeID, Valid: true})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	nbEleves, err := q.CountElevesPeriode(ctx, periodeID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	resp := exportSummaryResponse{
		PeriodeID: header.ID,
		Periode:   header.PeriodeName,
		Promo:     header.PromoName,
		Annee:     header.Annee,
		NbEleves:  nbEleves,
		Matieres:  make([]exportMatiereItem, 0, len(matieres)),
	}
	for _, m := range matieres {
		resp.Matieres = append(resp.Matieres, exportMatiereItem{
			ID:             m.ID,
			Name:           m.Name,
			NbSeances:      m.NbSeances,
			NbNonCloturees: m.NbNonCloturees,
			Premiere:       fmtJour(m.Premiere),
			Derniere:       fmtJour(m.Derniere),
		})
		// Étendue globale : bornes des matières, pour préremplir les champs de
		// dates du dialogue.
		if j := fmtJour(m.Premiere); j != "" && (resp.Du == "" || j < resp.Du) {
			resp.Du = j
		}
		if j := fmtJour(m.Derniere); j != "" && (resp.Au == "" || j > resp.Au) {
			resp.Au = j
		}
	}
	render.JSON(w, r, resp)
}

// fmtJour rend une date au format AAAA-MM-JJ, directement exploitable par un
// champ de saisie de date côté web. L'ordre lexicographique de ce format
// coïncide avec l'ordre chronologique, d'où les comparaisons ci-dessus.
func fmtJour(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(parisLoc).Format("2006-01-02")
}

// timestampCfg regroupe la configuration TSA et témoin passée aux handlers.
type timestampCfg struct {
	tsCfg      services.TimestampConfig
	caCertPath string
	witness    services.WitnessConfig
}
