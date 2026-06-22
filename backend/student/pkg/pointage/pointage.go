package pointage

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/pointage/gen"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var tokenSecret string

func SetTokenSecret(secret string) {
	tokenSecret = secret
}

type postPointageRequest struct {
	Code string `json:"code"`
}

type postPointageResponse struct {
	Matiere    string `json:"matiere"`
	SeanceID   int64  `json:"seance_id"`
	Statut     string `json:"statut"`
	PointeAt   string `json:"pointe_at"`
	DejaPointe bool   `json:"deja_pointe"`
}

func PostPointage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req postPointageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
			services.InvalidRequestError(w, r, "code requis", services.NO_INFORMATION, nil)
			return
		}

		userID := auth.GetSecurityUserId(r.Context())
		pgCtx := services.GetPgCtx(r.Context())
		q := gen.New(pgCtx.Db)
		ctx := context.Background()

		seance, err := resolveSeance(ctx, q, req.Code)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				services.InvalidRequestError(w, r, "séance introuvable", services.NO_INFORMATION, nil)
				return
			}
			if err.Error() == "token expiré" {
				services.InvalidRequestError(w, r, "token expiré", services.NO_INFORMATION, nil)
				return
			}
			if err.Error() == "signature invalide" || err.Error() == "token invalide" {
				services.InvalidRequestError(w, r, "token invalide", services.NO_INFORMATION, nil)
				return
			}
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		if seance.ClosedAt.Valid {
			services.InvalidRequestError(w, r, "séance fermée", services.NO_INFORMATION, nil)
			return
		}

		statut := "PRESENT"
		if time.Now().After(seance.OpenedAt.Time.Add(time.Duration(seance.LateAfterMinutes) * time.Minute)) {
			statut = "RETARD"
		}

		uid := int32(*userID)
		_, err = q.UpsertPointage(ctx, gen.UpsertPointageParams{
			SeanceID: seance.ID,
			UserID:   uid,
			Statut:   statut,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		dejaPointe := errors.Is(err, pgx.ErrNoRows)
		var pointeAt pgtype.Timestamptz
		if dejaPointe {
			existing, err2 := q.GetExistingPointage(ctx, gen.GetExistingPointageParams{
				SeanceID: seance.ID,
				UserID:   uid,
			})
			if err2 != nil {
				services.InternalServerError(w, r, err2.Error(), services.NO_INFORMATION, nil)
				return
			}
			statut = existing.Statut
			pointeAt = existing.PointeAt
		}

		resp := postPointageResponse{
			Matiere:    seance.MatiereName,
			SeanceID:   seance.ID,
			Statut:     statut,
			DejaPointe: dejaPointe,
		}
		if pointeAt.Valid {
			resp.PointeAt = pointeAt.Time.Format(time.RFC3339)
		} else {
			resp.PointeAt = time.Now().UTC().Format(time.RFC3339)
		}

		render.JSON(w, r, resp)
	}
}

type seanceRow = gen.GetSeanceByCodeRow

// resolveSeance accepte un code court ou un token HMAC signé par l'admin.
// Format token : base64url(JSON{seance_id,exp}).base64url(HMAC-SHA256)
func resolveSeance(ctx context.Context, q *gen.Queries, code string) (seanceRow, error) {
	if strings.Contains(code, ".") {
		seanceID, err := verifyToken(code)
		if err != nil {
			return seanceRow{}, err
		}
		row, err := q.GetSeanceByID(ctx, seanceID)
		if err != nil {
			return seanceRow{}, err
		}
		return seanceRow{
			ID:               row.ID,
			OpenedAt:         row.OpenedAt,
			ClosedAt:         row.ClosedAt,
			LateAfterMinutes: row.LateAfterMinutes,
			MatiereName:      row.MatiereName,
		}, nil
	}
	return q.GetSeanceByCode(ctx, code)
}

type tokenPayload struct {
	SeanceID int64 `json:"seance_id"`
	Exp      int64 `json:"exp"`
}

func verifyToken(token string) (int64, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return 0, errors.New("token invalide")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, errors.New("token invalide")
	}
	mac := hmac.New(sha256.New, []byte(tokenSecret))
	mac.Write(payloadJSON)
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return 0, errors.New("signature invalide")
	}
	var p tokenPayload
	if err := json.Unmarshal(payloadJSON, &p); err != nil {
		return 0, errors.New("token invalide")
	}
	if time.Now().Unix() > p.Exp {
		return 0, errors.New("token expiré")
	}
	return p.SeanceID, nil
}
