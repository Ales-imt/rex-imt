package evaluation

import (
	"back-rex-admin/pkg/ia"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"
)

// ─── Response types ───────────────────────────────────────────────────────────

type anneeAcademiqueResp struct {
	ID      int64   `json:"id"`
	Libelle string  `json:"libelle"`
	Debut   *string `json:"debut"`
	Fin     *string `json:"fin"`
	Active  bool    `json:"active"`
}

type matiereStatus struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	NbRepondants int    `json:"nb_repondants"`
	DotStatus    string `json:"dot_status"`
}

type periodeTree struct {
	ID       int64           `json:"id"`
	Name     string          `json:"name"`
	Matieres []matiereStatus `json:"matieres"`
}

type promotionTree struct {
	ID       int64         `json:"id"`
	Name     string        `json:"name"`
	Periodes []periodeTree `json:"periodes"`
}

type npsDistribution struct {
	PctDetracteurs float64 `json:"pct_detracteurs"`
	PctPassifs     float64 `json:"pct_passifs"`
	PctPromoteurs  float64 `json:"pct_promoteurs"`
}

type chipStat struct {
	ChipID    string  `json:"chip_id"`
	Libelle   string  `json:"libelle"`
	Dimension string  `json:"dimension"`
	Polarite  string  `json:"polarite"`
	Nb        int     `json:"nb"`
	Pct       float64 `json:"pct"`
}

type syntheseIAResp struct {
	ID            string  `json:"id"`
	MatiereID     int64   `json:"matiere_id"`
	Semestre      string  `json:"semestre"`
	NbRepondants  int     `json:"nb_repondants"`
	SyntheseTexte string  `json:"synthese_texte"`
	Statut        string  `json:"statut"`
	GeneratedAt   string  `json:"generated_at"`
	ValidatedBy   *string `json:"validated_by"`
	ValidatedAt   *string `json:"validated_at"`
}

type evalStatsResp struct {
	NbRepondants        int              `json:"nb_repondants"`
	ScorePeda           *float64         `json:"score_peda"`
	ScoreClarte         *float64         `json:"score_clarte"`
	ScoreContenu        *float64         `json:"score_contenu"`
	ScoreDifficulte     *float64         `json:"score_difficulte"`
	ScoreSupports       *float64         `json:"score_supports"`
	ScoreAmbiance       *float64         `json:"score_ambiance"`
	NpsMoyen            *float64         `json:"nps_moyen"`
	PctChargeLourde     float64          `json:"pct_charge_lourde"`
	PctChargeLegere     float64          `json:"pct_charge_legere"`
	PctFormatPresentiel float64          `json:"pct_format_presentiel"`
	PctFormatDistanciel float64          `json:"pct_format_distanciel"`
	PctFormatHybride    float64          `json:"pct_format_hybride"`
	PctTendanceBaisse   float64          `json:"pct_tendance_baisse"`
	PctTendanceStable   float64          `json:"pct_tendance_stable"`
	PctTendanceProgres  float64          `json:"pct_tendance_progres"`
	NpsDistribution     *npsDistribution `json:"nps_distribution"`
	ChipStats           []chipStat       `json:"chip_stats"`
	Synthese            *syntheseIAResp  `json:"synthese"`
}

type verbatimResp struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	Dimension string `json:"dimension"`
	Texte     string `json:"texte"`
	CreatedAt string `json:"created_at"`
}

type verbatimsPage struct {
	Data  []verbatimResp `json:"data"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func dotStatus(nb int) string {
	switch {
	case nb >= 5:
		return "OK"
	case nb >= 1:
		return "WARN"
	default:
		return "NONE"
	}
}

func ptrFloat(f pgtype.Float8) *float64 {
	if !f.Valid {
		return nil
	}
	v := f.Float64
	return &v
}

func fmtTimestamp(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

func uuidStr(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func syntheseFromRow(row GetLastSyntheseByMatiereRow) syntheseIAResp {
	s := syntheseIAResp{
		ID:            uuidStr(row.ID),
		MatiereID:     row.MatiereID,
		Semestre:      row.Semestre,
		NbRepondants:  int(row.NbRepondants),
		SyntheseTexte: row.SyntheseTexte,
		Statut:        row.Statut,
		GeneratedAt:   row.GeneratedAt.Time.Format(time.RFC3339),
		ValidatedAt:   fmtTimestamp(row.ValidatedAt),
	}
	if row.ValidatedBy.Valid {
		s.ValidatedBy = &row.ValidatedBy.String
	}
	return s
}

func verbatimFromRow(row GetVerbatimsByMatiereRow) verbatimResp {
	v := verbatimResp{
		ID:        uuidStr(row.ID),
		SessionID: uuidStr(row.SessionID),
		Dimension: row.Dimension,
		Texte:     row.Texte,
	}
	if row.CreatedAt.Valid {
		v.CreatedAt = row.CreatedAt.Time.Format(time.RFC3339)
	}
	return v
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

func GetAnnees(w http.ResponseWriter, r *http.Request) {
	q := New(services.GetPgCtx(r.Context()).Db)
	rows, err := q.GetAnneesAcademiques(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	result := make([]anneeAcademiqueResp, len(rows))
	for i, a := range rows {
		result[i] = anneeAcademiqueResp{
			ID:     a.ID,
			Active: a.Active,
			Debut:  fmtTimestamp(a.Debut),
			Fin:    fmtTimestamp(a.Fin),
		}
		if a.Libelle.Valid {
			result[i].Libelle = a.Libelle.String
		}
	}
	render.JSON(w, r, result)
}

func GetPromotionTree(w http.ResponseWriter, r *http.Request) {
	anneeIDStr := chi.URLParam(r, "anneeId")
	anneeID, err := strconv.ParseInt(anneeIDStr, 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "invalid anneeId", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	rows, err := q.GetPromotionTreeByAnnee(context.Background(), pgtype.Int8{Int64: anneeID, Valid: true})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	type periodeKey struct{ promoID, periodeID int64 }
	promos := []*promotionTree{}
	promoIdx := map[int64]int{}
	periodeIdx := map[periodeKey]int{}

	for _, row := range rows {
		promoName := row.PromotionName.String
		pi, ok := promoIdx[row.PromotionID]
		if !ok {
			promos = append(promos, &promotionTree{ID: row.PromotionID, Name: promoName, Periodes: []periodeTree{}})
			pi = len(promos) - 1
			promoIdx[row.PromotionID] = pi
		}

		pk := periodeKey{row.PromotionID, row.PeriodeID}
		peI, ok := periodeIdx[pk]
		if !ok {
			promos[pi].Periodes = append(promos[pi].Periodes, periodeTree{ID: row.PeriodeID, Name: row.PeriodeName, Matieres: []matiereStatus{}})
			peI = len(promos[pi].Periodes) - 1
			periodeIdx[pk] = peI
		}

		nb := int(row.NbRepondants)
		promos[pi].Periodes[peI].Matieres = append(promos[pi].Periodes[peI].Matieres, matiereStatus{
			ID:           row.MatiereID,
			Name:         row.MatiereName,
			NbRepondants: nb,
			DotStatus:    dotStatus(nb),
		})
	}

	result := make([]promotionTree, len(promos))
	for i, p := range promos {
		result[i] = *p
	}
	render.JSON(w, r, result)
}

func GetMatiereStats(w http.ResponseWriter, r *http.Request) {
	matiereIDStr := chi.URLParam(r, "matiereId")
	matiereID, err := strconv.ParseInt(matiereIDStr, 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "invalid matiereId", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	st, err := q.GetEvalStatsByMatiere(ctx, matiereID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	stats := evalStatsResp{
		NbRepondants:        int(st.NbRepondants),
		ScorePeda:           ptrFloat(st.ScorePeda),
		ScoreClarte:         ptrFloat(st.ScoreClarte),
		ScoreContenu:        ptrFloat(st.ScoreContenu),
		ScoreDifficulte:     ptrFloat(st.ScoreDifficulte),
		ScoreSupports:       ptrFloat(st.ScoreSupports),
		ScoreAmbiance:       ptrFloat(st.ScoreAmbiance),
		NpsMoyen:            ptrFloat(st.NpsMoyen),
		PctChargeLourde:     st.PctChargeLourde.Float64,
		PctChargeLegere:     st.PctChargeLegere.Float64,
		PctFormatPresentiel: st.PctFormatPresentiel.Float64,
		PctFormatDistanciel: st.PctFormatDistanciel.Float64,
		PctFormatHybride:    st.PctFormatHybride.Float64,
		PctTendanceBaisse:   st.PctTendanceBaisse.Float64,
		PctTendanceStable:   st.PctTendanceStable.Float64,
		PctTendanceProgres:  st.PctTendanceProgres.Float64,
	}

	nps, err := q.GetNpsDistributionByMatiere(ctx, matiereID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	stats.NpsDistribution = &npsDistribution{
		PctDetracteurs: nps.PctDetracteurs.Float64,
		PctPassifs:     nps.PctPassifs.Float64,
		PctPromoteurs:  nps.PctPromoteurs.Float64,
	}

	chips, err := q.GetChipStatsByMatiere(ctx, matiereID)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	stats.ChipStats = make([]chipStat, len(chips))
	for i, c := range chips {
		stats.ChipStats[i] = chipStat{
			ChipID:    c.ChipID,
			Libelle:   c.Libelle,
			Dimension: c.Dimension,
			Polarite:  c.Polarite,
			Nb:        int(c.Nb),
			Pct:       c.Pct.Float64,
		}
	}

	syn, err := q.GetLastSyntheseByMatiere(ctx, matiereID)
	if err == nil {
		s := syntheseFromRow(syn)
		stats.Synthese = &s
	} else if !errors.Is(err, pgx.ErrNoRows) {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, stats)
}

func GetVerbatims(w http.ResponseWriter, r *http.Request) {
	matiereIDStr := chi.URLParam(r, "matiereId")
	matiereID, err := strconv.ParseInt(matiereIDStr, 10, 64)
	if err != nil {
		services.InvalidRequestError(w, r, "invalid matiereId", services.NO_INFORMATION, nil)
		return
	}

	page := 1
	limit := 10
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	dimension := r.URL.Query().Get("dimension")
	offset := (page - 1) * limit

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	var rows []GetVerbatimsByMatiereRow
	if dimension != "" {
		var filtered []GetVerbatimsByMatiereFilteredRow
		filtered, err = q.GetVerbatimsByMatiereFiltered(ctx, GetVerbatimsByMatiereFilteredParams{
			MatiereID: matiereID,
			Dimension: dimension,
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
		if err == nil {
			rows = make([]GetVerbatimsByMatiereRow, len(filtered))
			for i, r := range filtered {
				rows[i] = GetVerbatimsByMatiereRow(r)
			}
		}
	} else {
		rows, err = q.GetVerbatimsByMatiere(ctx, GetVerbatimsByMatiereParams{
			MatiereID: matiereID,
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	data := make([]verbatimResp, len(rows))
	for i, row := range rows {
		data[i] = verbatimFromRow(row)
	}
	render.JSON(w, r, verbatimsPage{Data: data, Page: page, Limit: limit})
}

func GenerateSynthese(connector ia.IAConnector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		matiereIDStr := chi.URLParam(r, "matiereId")
		matiereID, err := strconv.ParseInt(matiereIDStr, 10, 64)
		if err != nil {
			services.InvalidRequestError(w, r, "invalid matiereId", services.NO_INFORMATION, nil)
			return
		}

		q := New(services.GetPgCtx(r.Context()).Db)
		ctx := context.Background()

		var (
			evalRow  GetEvalDataForPromptRow
			chipRows []GetChipsForPromptRow
			verbRows []GetVerbatimsForPromptRow
		)

		g, gCtx := errgroup.WithContext(ctx)
		g.Go(func() error {
			var e error
			evalRow, e = q.GetEvalDataForPrompt(gCtx, matiereID)
			return e
		})
		g.Go(func() error {
			var e error
			chipRows, e = q.GetChipsForPrompt(gCtx, matiereID)
			return e
		})
		g.Go(func() error {
			var e error
			verbRows, e = q.GetVerbatimsForPrompt(gCtx, GetVerbatimsForPromptParams{
				MatiereID: matiereID,
				Limit:     20,
			})
			return e
		})
		if err := g.Wait(); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		chips := make([]PromptChip, len(chipRows))
		for i, c := range chipRows {
			chips[i] = PromptChip{Libelle: c.Libelle, Polarite: c.Polarite, Nb: int(c.Nb)}
		}
		verbs := make([]PromptVerbatim, len(verbRows))
		for i, v := range verbRows {
			verbs[i] = PromptVerbatim{Dimension: v.Dimension, Texte: v.Texte}
		}

		promoName := evalRow.PromotionName.String
		prompt := BuildPrompt(PromptData{
			Eval: PromptEvalData{
				MatiereName:   evalRow.MatiereName,
				PromotionName: promoName,
				PeriodeName:   evalRow.PeriodeName,
				NbRepondants:  int(evalRow.NbRepondants),
				ScorePeda:     ptrFloat(evalRow.ScorePeda),
				ScoreClarte:   ptrFloat(evalRow.ScoreClarte),
				ScoreContenu:  ptrFloat(evalRow.ScoreContenu),
				ScoreSupports: ptrFloat(evalRow.ScoreSupports),
				ScoreAmbiance: ptrFloat(evalRow.ScoreAmbiance),
				NpsMoyen:      ptrFloat(evalRow.NpsMoyen),
			},
			Chips:     chips,
			Verbatims: verbs,
		})

		text, err := connector.Analyze(ctx, prompt)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		inserted, err := q.InsertSynthese(ctx, InsertSyntheseParams{
			MatiereID:     matiereID,
			Semestre:      evalRow.PeriodeName,
			NbRepondants:  evalRow.NbRepondants,
			SyntheseTexte: text,
		})
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		render.JSON(w, r, syntheseIAResp{
			ID:            uuidStr(inserted.ID),
			MatiereID:     inserted.MatiereID,
			Semestre:      inserted.Semestre,
			NbRepondants:  int(inserted.NbRepondants),
			SyntheseTexte: inserted.SyntheseTexte,
			Statut:        inserted.Statut,
			GeneratedAt:   inserted.GeneratedAt.Time.Format(time.RFC3339),
		})
	}
}

func UpdateSyntheseStatut(w http.ResponseWriter, r *http.Request) {
	syntheseIDStr := chi.URLParam(r, "syntheseId")

	var body struct {
		Statut      string `json:"statut"`
		ValidatedBy string `json:"validated_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		services.InvalidRequestError(w, r, "corps invalide", services.NO_INFORMATION, nil)
		return
	}
	if body.Statut != "VALIDEE" && body.Statut != "ARCHIVEE" {
		services.InvalidRequestError(w, r, "statut invalide (VALIDEE ou ARCHIVEE attendu)", services.NO_INFORMATION, nil)
		return
	}

	var uid pgtype.UUID
	if err := uid.Scan(syntheseIDStr); err != nil {
		services.InvalidRequestError(w, r, "syntheseId invalide", services.NO_INFORMATION, nil)
		return
	}

	var validatedBy pgtype.Text
	if body.ValidatedBy != "" {
		validatedBy = pgtype.Text{String: body.ValidatedBy, Valid: true}
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	if err := q.UpdateSyntheseStatut(context.Background(), UpdateSyntheseStatutParams{
		ID:          uid,
		Statut:      body.Statut,
		ValidatedBy: validatedBy,
	}); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
