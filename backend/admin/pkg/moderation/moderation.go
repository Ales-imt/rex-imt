package moderation

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxContentLen = 2000
	maxReasonLen  = 500
)

// PendingItem : un feedback en attente de modération. On n'expose que le texte
// brut, l'horodatage et la promotion (qui sert à filtrer la file) — aucune
// donnée d'identification (ni strongbox, ni pseudo).
type PendingItem struct {
	ID         int32  `json:"id"`
	RawContent string `json:"raw_content"`
	CreatedAt  string `json:"created_at"`
	Promotion  string `json:"promotion,omitempty"`
}

// ListPending renvoie les feedbacks PENDING avec leur texte brut.
func ListPending(w http.ResponseWriter, r *http.Request) {
	pool := services.GetPgCtx(r.Context()).Db
	rows, err := New(pool).ListPendingModeration(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	out := make([]PendingItem, 0, len(rows))
	for _, row := range rows {
		item := PendingItem{ID: row.ID}
		if row.RawContent.Valid {
			item.RawContent = row.RawContent.String
		}
		if row.Promotion.Valid {
			item.Promotion = row.Promotion.String
		}
		if row.CreatedAt.Valid {
			item.CreatedAt = row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		out = append(out, item)
	}

	render.JSON(w, r, out)
}

type approveRequest struct {
	ContentModere string `json:"content_modere"`
}

// Approve publie un feedback : pose le contenu modéré, passe à PUBLISHED, trace
// le modérateur et efface raw_content, dans un unique UPDATE.
func Approve(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var input approveRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if utf8.RuneCountInString(input.ContentModere) > maxContentLen {
		services.InvalidRequestError(w, r, "content trop long", services.NO_INFORMATION, nil)
		return
	}

	moderatorID := *auth.GetSecurityUserId(r.Context())

	pool := services.GetPgCtx(r.Context()).Db
	n, err := New(pool).ApproveFeedback(context.Background(), ApproveFeedbackParams{
		ID:          id,
		Column2:     input.ContentModere,
		ModeratedBy: pgtype.Int4{Int32: int32(moderatorID), Valid: true},
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if n == 0 {
		services.InvalidRequestError(w, r, "feedback introuvable ou déjà modéré", services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

// makeReject refuse un feedback : passe à REJECTED avec motif, et remplace le
// texte brut par sa version chiffrée (age) — le contenu refusé n'est plus
// lisible en clair au repos. Il est ensuite purgé (cf. rgpd).
func makeReject(agePublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}

		var input rejectRequest
		if err := render.DecodeJSON(r.Body, &input); err != nil {
			services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		if utf8.RuneCountInString(input.Reason) > maxReasonLen {
			services.InvalidRequestError(w, r, "motif trop long", services.NO_INFORMATION, nil)
			return
		}

		moderatorID := *auth.GetSecurityUserId(r.Context())

		ctx := context.Background()
		tx, err := services.GetPgCtx(r.Context()).Db.Begin(ctx)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		defer tx.Rollback(ctx)
		q := New(tx)

		// Récupère le texte brut en attente pour le chiffrer avant de le refuser.
		raw, err := q.GetPendingRawContent(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			services.InvalidRequestError(w, r, "feedback introuvable ou déjà modéré", services.NO_INFORMATION, nil)
			return
		}
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		var encrypted pgtype.Text
		if raw != "" {
			cipher, err := services.EncryptAge(agePublicKey, raw)
			if err != nil {
				services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
				return
			}
			encrypted = services.ToPgText(cipher)
		}

		n, err := q.RejectFeedback(ctx, RejectFeedbackParams{
			ID:              id,
			RejectionReason: services.ToPgText(input.Reason),
			ModeratedBy:     pgtype.Int4{Int32: int32(moderatorID), Valid: true},
			RawContent:      encrypted,
		})
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		if n == 0 {
			services.InvalidRequestError(w, r, "feedback introuvable ou déjà modéré", services.NO_INFORMATION, nil)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func parseID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		services.InvalidRequestError(w, r, "id invalide", services.NO_INFORMATION, nil)
		return 0, false
	}
	return int32(id), true
}
