package feedback

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/season"
	"back-rex-eleve/pkg/student"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/go-chi/render"
)

type FeedbackRequest struct {
	Content string `json:"content"`
}

type FeedbackResponse struct {
	Message      string `json:"message"`
	Elo          int    `json:"elo"`
	Rank         int    `json:"rank"`
	EloGain      int    `json:"eloGain"`
	SeasonEndsIn string `json:"seasonEndsIn"`
}

func makeFeedbackHandler(agePublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var inputs []FeedbackRequest
		if err := render.DecodeJSON(r.Body, &inputs); err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		response, err := ProcessFeedback(r, inputs, agePublicKey)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		render.JSON(w, r, response)
	}
}

// 🔵 GET /feedback — récupère les données pour ThankView
func GetLastEloGain(w http.ResponseWriter, r *http.Request) {
	studentID := *auth.GetSecurityUserId(r.Context())
	pgCtx := services.GetPgCtx(r.Context())

	seasonID, err := season.GetOrCreateCurrentSeason(r)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	studentQueries := student.New(pgCtx.Db)
	seasonQueries := season.New(pgCtx.Db)

	elo, err := studentQueries.GetStudentElo(context.Background(), int32(studentID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	rank, err := studentQueries.GetStudentRank(context.Background(), int32(studentID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	eloGain, err := studentQueries.GetStudentLastEloGain(context.Background(), int32(studentID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	seasonEnd, err := seasonQueries.GetSeasonEndDate(context.Background(), int32(seasonID))
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, FeedbackResponse{
		Elo:          int(elo),
		EloGain:      int(eloGain.Int32),
		Rank:         int(rank),
		SeasonEndsIn: time.Until(seasonEnd.Time).Round(time.Minute).String(),
	})
}

// 🧠 ProcessFeedback : logique complète utilisée par POST
func ProcessFeedback(r *http.Request, inputs []FeedbackRequest, agePublicKey string) (*FeedbackResponse, error) {
	studentID := *auth.GetSecurityUserId(r.Context())
	pgCtx := services.GetPgCtx(r.Context())
	studentQueries := student.New(pgCtx.Db)
	seasonQueries := season.New(pgCtx.Db)

	seasonID, err := season.GetOrCreateCurrentSeason(r)
	if err != nil {
		return nil, err
	}

	if err := InsertFeedbacks(r, studentID, inputs, int32(seasonID), agePublicKey); err != nil {
		return nil, err
	}

	eloGain := 5 * len(inputs)

	err = studentQueries.ClearLastEloGain(context.Background(), int32(studentID))
	if err != nil {
		return nil, fmt.Errorf("échec reset elo gain : %w", err)
	}

	err = studentQueries.UpdateStudentElo(context.Background(), student.UpdateStudentEloParams{
		UserID:      int32(studentID),
		LastEloGain: services.ToPgInt4(eloGain),
	})
	if err != nil {
		return nil, err
	}

	elo, err := studentQueries.GetStudentElo(context.Background(), int32(studentID))
	if err != nil {
		return nil, err
	}

	rank, err := studentQueries.GetStudentRank(context.Background(), int32(studentID))
	if err != nil {
		return nil, err
	}

	seasonEnd, err := seasonQueries.GetSeasonEndDate(context.Background(), int32(seasonID))
	if err != nil {
		return nil, err
	}

	return &FeedbackResponse{
		Elo:          int(elo),
		EloGain:      eloGain,
		Rank:         int(rank),
		SeasonEndsIn: time.Until(seasonEnd.Time).Round(time.Minute).String(),
	}, nil
}

// 🧾 Insertion des feedbacks
func InsertFeedbacks(r *http.Request, studentID int, inputs []FeedbackRequest, seasonID int32, agePublicKey string) error {
	pgCtx := services.GetPgCtx(r.Context())
	queries := New(pgCtx.Db)
	now := time.Now()

	strongbox, err := encryptStrongbox(agePublicKey, r.RemoteAddr, studentID)
	if err != nil {
		return fmt.Errorf("échec chiffrement strongbox : %w", err)
	}

	for _, input := range inputs {
		_, err := queries.InsertFeedbacks(context.Background(), InsertFeedbacksParams{
			UserID:    services.ToPgInt4(studentID),
			Content:   input.Content,
			CreatedAt: services.ToPgTimestamptz(&now),
			SeasonID:  services.ToPgInt4(int(seasonID)),
			Strongbox: services.ToPgText(strongbox),
		})
		if err != nil {
			return fmt.Errorf("échec insertion feedback : %w", err)
		}
	}
	return nil
}

// encryptStrongbox chiffre l'IP et l'ID élève avec la clef publique age fournie.
func encryptStrongbox(publicKeyStr string, ip string, studentID int) (string, error) {
	recipient, err := age.ParseX25519Recipient(publicKeyStr)
	if err != nil {
		return "", fmt.Errorf("clef publique age invalide : %w", err)
	}

	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)
	w, err := age.Encrypt(armorWriter, recipient)
	if err != nil {
		return "", fmt.Errorf("age encrypt : %w", err)
	}

	payload := fmt.Sprintf("ip=%s student_id=%d", ip, studentID)
	if _, err := w.Write([]byte(payload)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	if err := armorWriter.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
