package moderation

// Modération des verbatims d'évaluation (eval_verbatim). Décalque exactement le
// pipeline des feedbacks (cf. moderation.go) : raw_texte est relu par un
// modérateur, puis publié dans texte ou remplacé par sa version chiffrée (age)
// en cas de refus. Tant qu'un verbatim est PENDING, il n'est ni affiché dans
// l'admin ni envoyé à l'IA.
//
// Seule différence structurelle : la clé d'eval_verbatim est un uuid.

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PendingVerbatim : un verbatim en attente de modération. On n'expose que le
// texte brut, sa dimension, l'horodatage et la promotion du cours évalué (qui
// sert à filtrer la file) — jamais session_id, qui relierait le verbatim à son
// auteur via eval_session.
type PendingVerbatim struct {
	ID        string `json:"id"`
	RawTexte  string `json:"raw_texte"`
	Dimension string `json:"dimension"`
	CreatedAt string `json:"created_at"`
	Promotion string `json:"promotion,omitempty"`
}

// ListPendingVerbatim renvoie les verbatims PENDING avec leur texte brut.
func ListPendingVerbatim(w http.ResponseWriter, r *http.Request) {
	pool := services.GetPgCtx(r.Context()).Db
	rows, err := New(pool).ListPendingVerbatims(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	out := make([]PendingVerbatim, 0, len(rows))
	for _, row := range rows {
		item := PendingVerbatim{
			ID:        uuidStr(row.ID),
			Dimension: row.Dimension,
			Promotion: row.Promotion,
		}
		if row.RawTexte.Valid {
			item.RawTexte = row.RawTexte.String
		}
		if row.CreatedAt.Valid {
			item.CreatedAt = row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		out = append(out, item)
	}

	render.JSON(w, r, out)
}

type approveVerbatimRequest struct {
	TexteModere string `json:"texte_modere"`
}

// ApproveVerbatim publie un verbatim : pose le texte modéré, passe à PUBLISHED,
// trace le modérateur et efface raw_texte, dans un unique UPDATE.
func ApproveVerbatim(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	var input approveVerbatimRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if utf8.RuneCountInString(input.TexteModere) > maxContentLen {
		services.InvalidRequestError(w, r, "texte trop long", services.NO_INFORMATION, nil)
		return
	}

	moderatorID := *auth.GetSecurityUserId(r.Context())

	pool := services.GetPgCtx(r.Context()).Db
	n, err := New(pool).ApproveVerbatim(context.Background(), ApproveVerbatimParams{
		ID:          id,
		Column2:     input.TexteModere,
		ModeratedBy: pgtype.Int4{Int32: int32(moderatorID), Valid: true},
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if n == 0 {
		services.InvalidRequestError(w, r, "verbatim introuvable ou déjà modéré", services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// makeRejectVerbatim refuse un verbatim : passe à REJECTED avec motif, et
// remplace le texte brut par sa version chiffrée (age) — le contenu refusé
// n'est plus lisible en clair au repos. Il est ensuite purgé (cf. rgpd).
func makeRejectVerbatim(agePublicKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseUUID(w, r)
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
		raw, err := q.GetPendingRawVerbatim(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			services.InvalidRequestError(w, r, "verbatim introuvable ou déjà modéré", services.NO_INFORMATION, nil)
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

		n, err := q.RejectVerbatim(ctx, RejectVerbatimParams{
			ID:              id,
			RejectionReason: services.ToPgText(input.Reason),
			ModeratedBy:     pgtype.Int4{Int32: int32(moderatorID), Valid: true},
			RawTexte:        encrypted,
		})
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		if n == 0 {
			services.InvalidRequestError(w, r, "verbatim introuvable ou déjà modéré", services.NO_INFORMATION, nil)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func parseUUID(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	var id pgtype.UUID
	if err := id.Scan(chi.URLParam(r, "id")); err != nil {
		services.InvalidRequestError(w, r, "id invalide", services.NO_INFORMATION, nil)
		return id, false
	}
	return id, true
}

func uuidStr(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
