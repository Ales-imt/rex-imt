package evaluation

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/evaluation/gen"
	"back-rex-eleve/pkg/programme"
	"back-rex-eleve/pkg/service"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SubmitEvaluationRequest reflète exactement les 7 écrans du formulaire.
// Les chips et verbatims sont optionnels (omitempty).
type SubmitEvaluationRequest struct {
	// Écran 0 — contexte
	MatiereID        int64  `json:"matiere_id,string"`
	FormatSuivi      string `json:"format_suivi"`      // PRESENTIEL | DISTANCIEL | HYBRIDE
	TendanceSemestre string `json:"tendance_semestre"` // PROGRES | STABLE | BAISSE
	Assiduite        string `json:"assiduite"`         // TOUTES | QUELQUES_ABSENCES | BEAUCOUP

	// Étape 1 — pédagogie
	ScorePedaGlobal int32 `json:"score_peda_global"`
	ScorePedaClarte int32 `json:"score_peda_clarte"`

	// Étape 2 — contenu
	ScoreContenuAdequation int32    `json:"score_contenu_adequation"`
	ChipsContenu           []string `json:"chips_contenu,omitempty"`

	// Étape 3 — charge
	ChargePerception    string `json:"charge_perception"`     // LEGERE | ADAPTEE | LOURDE
	ChargeHeuresSemaine string `json:"charge_heures_semaine"` // H_MOINS_1 | H_1_3 | H_3_5 | H_PLUS_5
	ScoreDifficulte     int32  `json:"score_difficulte"`

	// Étape 4 — supports
	ScoreSupports    int32    `json:"score_supports"`
	ChipsSupports    []string `json:"chips_supports,omitempty"`
	VerbatimSupports string   `json:"verbatim_supports,omitempty"`

	// Étape 5 — ambiance
	ScoreAmbiance    int32    `json:"score_ambiance"`
	ChipsAmbiance    []string `json:"chips_ambiance,omitempty"`
	VerbatimAmbiance string   `json:"verbatim_ambiance,omitempty"`

	// Étape 6 — NPS
	Nps         int32  `json:"nps"`
	VerbatimNps string `json:"verbatim_nps,omitempty"`
}

type MatiereSummary struct {
	MatiereID string `json:"matiereId"`
	Nom       string `json:"nom"`
	Prof      string `json:"prof"`
	Formation string `json:"formation"`
	Fait      bool   `json:"fait"`
}

type matiereGroup struct {
	cocle     string
	nom       string
	prof      string
	formation string
	lastDate  string // YYYY-MM-DD
	lastHF    string // HH:MM
}

func schoolYearDates(now time.Time) (start, end string) {
	year := now.Year()
	if now.Month() < time.September {
		year--
	}
	s := time.Date(year, time.September, 1, 0, 0, 0, 0, time.UTC)
	e := time.Date(year+1, time.July, 1, 0, 0, 0, 0, time.UTC)
	return s.Format("20060102"), e.Format("20060102")
}

// sessionDone retourne true si la séance (date YYYY-MM-DD + fin HH:MM) est terminée.
func sessionDone(date, hf string, now time.Time) bool {
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+hf, now.Location())
	if err != nil {
		return false
	}
	return !t.After(now)
}

func getMatiere(connector programme.ProgrammeConnector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pseudo := r.Header.Get("X-Anon-Id")
		ctx := context.Background()

		userID := auth.GetSecurityUserId(r.Context())
		pgCtx := services.GetPgCtx(r.Context())

		user, err := auth.New(pgCtx.Db).GetUserById(ctx, int32(*userID))
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		start, end := schoolYearDates(time.Now())
		seances, err := connector.GetProgramme(r.Context(), user.Email, start, end)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		// Regrouper les séances par COCLE et retenir la dernière séance de chaque matière
		groups := make(map[string]*matiereGroup)
		for _, s := range seances {
			if s.Cocle == "" {
				continue
			}
			g, ok := groups[s.Cocle]
			if !ok {
				g = &matiereGroup{cocle: s.Cocle, nom: s.Cours, prof: s.Prof, formation: s.Promo}
				groups[s.Cocle] = g
			}
			// Garder la séance la plus tardive (tri lexicographique valide sur YYYY-MM-DD HH:MM)
			if s.Date > g.lastDate || (s.Date == g.lastDate && s.HF > g.lastHF) {
				g.lastDate = s.Date
				g.lastHF = s.HF
				g.prof = s.Prof
				g.formation = s.Promo
			}
		}

		// Ne retenir que les matières dont toutes les séances sont terminées (dernière séance passée)
		now := time.Now()
		eligible := make(map[string]*MatiereSummary)
		for cocle, g := range groups {
			if sessionDone(g.lastDate, g.lastHF, now) {
				eligible[cocle] = &MatiereSummary{
					MatiereID: cocle,
					Nom:       g.nom,
					Prof:      g.prof,
					Formation: g.formation,
				}
			}
		}

		if pseudo != "" {
			queries := gen.New(pgCtx.Db)
			submittedIDs, err := queries.GetSubmittedMatiereIDs(ctx, pseudo)
			if err == nil {
				for _, id := range submittedIDs {
					if m, ok := eligible[strconv.FormatInt(id, 10)]; ok {
						m.Fait = true
					}
				}
			}
		}

		result := make([]MatiereSummary, 0, len(eligible))
		for _, m := range eligible {
			result = append(result, *m)
		}
		sort.Slice(result, func(i, j int) bool {
			return result[i].Nom < result[j].Nom
		})

		render.JSON(w, r, result)
	}
}

func SubmitEvaluation(agePublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Identifiant anonyme transmis par le client (jamais lié au serveur d'auth)
		pseudo := r.Header.Get("X-Anon-Id")
		if pseudo == "" {
			services.InvalidRequestError(w, r, "pseudo requis", services.NO_INFORMATION, nil)
			return
		}

		var req SubmitEvaluationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			services.InvalidRequestError(w, r, "payload invalide", services.NO_INFORMATION, nil)
			return
		}

		if err := validateRequest(&req); err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		pgCtx := services.GetPgCtx(r.Context())
		queries := gen.New(pgCtx.Db)
		ctx := context.Background()

		// Vérification anti-doublon avant toute insertion
		alreadySubmitted, err := queries.EvalSessionExists(ctx, gen.EvalSessionExistsParams{
			Pseudo:    pseudo,
			MatiereID: req.MatiereID,
		})
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		if alreadySubmitted {
			services.InvalidRequestError(w, r, "évaluation déjà soumise pour ce cours ce semestre", services.NO_INFORMATION, nil)
			return
		}

		// Toutes les insertions dans une transaction :
		// si une étape échoue, rien n'est persisté
		tx, err := pgCtx.Db.Begin(ctx)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		defer tx.Rollback(ctx)

		qtx := queries.WithTx(tx)

		// 1 — Session (écran 0)
		sessionID, err := qtx.InsertEvalSession(ctx, gen.InsertEvalSessionParams{
			Pseudo:           pseudo,
			MatiereID:        req.MatiereID,
			FormatSuivi:      req.FormatSuivi,
			TendanceSemestre: req.TendanceSemestre,
			Assiduite:        req.Assiduite,
		})
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		// 2 — Scores (étapes 1 à 6)
		if err := qtx.InsertEvalScores(ctx, gen.InsertEvalScoresParams{
			SessionID:              sessionID,
			ScorePedaGlobal:        pgtype.Int2{Int16: int16(req.ScorePedaGlobal), Valid: true},
			ScorePedaClarte:        pgtype.Int2{Int16: int16(req.ScorePedaClarte), Valid: true},
			ScoreContenuAdequation: pgtype.Int2{Int16: int16(req.ScoreContenuAdequation), Valid: true},
			ChargePerception:       pgtype.Text{String: req.ChargePerception, Valid: true},
			ChargeHeuresSemaine:    pgtype.Text{String: req.ChargeHeuresSemaine, Valid: true},
			ScoreDifficulte:        pgtype.Int2{Int16: int16(req.ScoreDifficulte), Valid: true},
			ScoreSupports:          pgtype.Int2{Int16: int16(req.ScoreSupports), Valid: true},
			ScoreAmbiance:          pgtype.Int2{Int16: int16(req.ScoreAmbiance), Valid: true},
			Nps:                    pgtype.Int2{Int16: int16(req.Nps), Valid: true},
		}); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		// 3 — Chips (toutes dimensions confondues)
		allChips := mergeChips(req.ChipsContenu, req.ChipsSupports, req.ChipsAmbiance)
		for _, chipID := range allChips {
			if err := qtx.InsertEvalChipReponse(ctx, gen.InsertEvalChipReponseParams{
				SessionID: sessionID,
				ChipID:    chipID,
			}); err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}
		}

		// 4 — Verbatims optionnels
		userID := auth.GetSecurityUserId(r.Context())
		studentID := *userID
		strongbox, err := service.EncryptStrongbox(agePublicKey, r.RemoteAddr, studentID)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		verbatims := []struct {
			dimension string
			texte     string
		}{
			{"SUPPORTS", req.VerbatimSupports},
			{"AMBIANCE", req.VerbatimAmbiance},
			{"NPS", req.VerbatimNps},
		}
		for _, v := range verbatims {
			if v.texte == "" {
				continue
			}
			if err := qtx.InsertEvalVerbatim(ctx, gen.InsertEvalVerbatimParams{
				SessionID: sessionID,
				Dimension: v.dimension,
				Texte:     v.texte,
				Strongbox: services.ToPgText(strongbox),
			}); err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}
		}

		// 5 — Marquage de la session comme soumise
		if err := qtx.SubmitEvalSession(ctx, sessionID); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		// Commit — tout ou rien
		if err := tx.Commit(ctx); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		render.JSON(w, r, map[string]string{"session_id": sessionID.String()})
	}
}

// validateRequest vérifie les champs obligatoires et les valeurs autorisées.
func validateRequest(req *SubmitEvaluationRequest) error {
	if req.MatiereID == 0 {
		return errorf("matiere_id est requis")
	}

	validFormats := map[string]bool{"PRESENTIEL": true, "DISTANCIEL": true, "HYBRIDE": true}
	if !validFormats[req.FormatSuivi] {
		return errorf("format_suivi invalide : %s", req.FormatSuivi)
	}

	validTendances := map[string]bool{"PROGRES": true, "STABLE": true, "BAISSE": true}
	if !validTendances[req.TendanceSemestre] {
		return errorf("tendance_semestre invalide : %s", req.TendanceSemestre)
	}

	validAssiduite := map[string]bool{"TOUTES": true, "QUELQUES_ABSENCES": true, "BEAUCOUP": true}
	if !validAssiduite[req.Assiduite] {
		return errorf("assiduite invalide : %s", req.Assiduite)
	}

	validChargePerception := map[string]bool{"LEGERE": true, "ADAPTEE": true, "LOURDE": true}
	if !validChargePerception[req.ChargePerception] {
		return errorf("charge_perception invalide : %s", req.ChargePerception)
	}

	validChargeHeures := map[string]bool{"H_MOINS_1": true, "H_1_3": true, "H_3_5": true, "H_PLUS_5": true}
	if !validChargeHeures[req.ChargeHeuresSemaine] {
		return errorf("charge_heures_semaine invalide : %s", req.ChargeHeuresSemaine)
	}

	if err := checkScore("score_peda_global", req.ScorePedaGlobal, 1, 5); err != nil {
		return err
	}
	if err := checkScore("score_peda_clarte", req.ScorePedaClarte, 1, 5); err != nil {
		return err
	}
	if err := checkScore("score_contenu_adequation", req.ScoreContenuAdequation, 1, 5); err != nil {
		return err
	}
	if err := checkScore("score_difficulte", req.ScoreDifficulte, 1, 5); err != nil {
		return err
	}
	if err := checkScore("score_supports", req.ScoreSupports, 1, 5); err != nil {
		return err
	}
	if err := checkScore("score_ambiance", req.ScoreAmbiance, 1, 5); err != nil {
		return err
	}
	if err := checkScore("nps", req.Nps, 0, 10); err != nil {
		return err
	}

	return nil
}

// mergeChips regroupe toutes les chips de toutes les dimensions.
func mergeChips(slices ...[]string) []string {
	var result []string
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// checkScore vérifie qu'un score est dans la plage [min, max].
func checkScore(field string, val, min, max int32) error {
	if val < min || val > max {
		return errorf("%s doit être entre %d et %d, reçu : %d", field, min, max, val)
	}
	return nil
}

// errorf est un raccourci local pour fmt.Errorf sans importer fmt au niveau du package.
func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

type SessionDetailResponse struct {
	SessionID              string `json:"session_id"`
	FormatSuivi            string `json:"format_suivi"`
	TendanceSemestre       string `json:"tendance_semestre"`
	Assiduite              string `json:"assiduite"`
	ScorePedaGlobal        int16  `json:"score_peda_global"`
	ScorePedaClarte        int16  `json:"score_peda_clarte"`
	ScoreContenuAdequation int16  `json:"score_contenu_adequation"`
	ChargePerception       string `json:"charge_perception"`
	ChargeHeuresSemaine    string `json:"charge_heures_semaine"`
	ScoreDifficulte        int16  `json:"score_difficulte"`
	ScoreSupports          int16  `json:"score_supports"`
	ScoreAmbiance          int16  `json:"score_ambiance"`
	Nps                    int16  `json:"nps"`
}

func getSessionDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pseudo := r.Header.Get("X-Anon-Id")
		if pseudo == "" {
			services.InvalidRequestError(w, r, "pseudo requis", services.NO_INFORMATION, nil)
			return
		}

		matiereIDStr := r.URL.Query().Get("matiere_id")
		matiereID, err := strconv.ParseInt(matiereIDStr, 10, 64)
		if err != nil {
			services.InvalidRequestError(w, r, "matiere_id invalide", services.NO_INFORMATION, nil)
			return
		}

		pgCtx := services.GetPgCtx(r.Context())
		queries := gen.New(pgCtx.Db)
		ctx := context.Background()

		row, err := queries.GetSessionByMatiere(ctx, gen.GetSessionByMatiereParams{
			Pseudo:    pseudo,
			MatiereID: matiereID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "session introuvable", http.StatusNotFound)
				return
			}
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		render.JSON(w, r, SessionDetailResponse{
			SessionID:              row.ID.String(),
			FormatSuivi:            row.FormatSuivi,
			TendanceSemestre:       row.TendanceSemestre,
			Assiduite:              row.Assiduite,
			ScorePedaGlobal:        row.ScorePedaGlobal.Int16,
			ScorePedaClarte:        row.ScorePedaClarte.Int16,
			ScoreContenuAdequation: row.ScoreContenuAdequation.Int16,
			ChargePerception:       row.ChargePerception.String,
			ChargeHeuresSemaine:    row.ChargeHeuresSemaine.String,
			ScoreDifficulte:        row.ScoreDifficulte.Int16,
			ScoreSupports:          row.ScoreSupports.Int16,
			ScoreAmbiance:          row.ScoreAmbiance.Int16,
			Nps:                    row.Nps.Int16,
		})
	}
}

func deleteSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pseudo := r.Header.Get("X-Anon-Id")
		if pseudo == "" {
			services.InvalidRequestError(w, r, "pseudo requis", services.NO_INFORMATION, nil)
			return
		}

		sessionIDStr := chi.URLParam(r, "sessionId")
		var sessionID pgtype.UUID
		if err := sessionID.Scan(sessionIDStr); err != nil {
			services.InvalidRequestError(w, r, "sessionId invalide", services.NO_INFORMATION, nil)
			return
		}

		pgCtx := services.GetPgCtx(r.Context())
		queries := gen.New(pgCtx.Db)
		ctx := context.Background()

		if err := queries.DeleteEvalSession(ctx, gen.DeleteEvalSessionParams{
			SessionID: sessionID,
			Pseudo:    pseudo,
		}); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
