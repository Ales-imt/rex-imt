package annee

import (
	"back-rex-common/pkg/services"
	"net/http"

	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
)

func validateAnnee(input CreateAnneeParams) []services.FormValidation {
	issues := []services.FormValidation{}

	if input.Name == "" {
		issues = append(issues, services.FormValidation{
			Path:    "name",
			Message: "Nom obligatoire",
		})
	}

	if !input.Debut.Valid {
		issues = append(issues, services.FormValidation{
			Path:    "debut",
			Message: "Date de début obligatoire",
		})
	}

	if !input.Fin.Valid {
		issues = append(issues, services.FormValidation{
			Path:    "fin",
			Message: "Date de fin obligatoire",
		})
	}

	if input.Debut.Valid && input.Fin.Valid && !input.Fin.Time.After(input.Debut.Time) {
		issues = append(issues, services.FormValidation{
			Path:    "fin",
			Message: "La date de fin doit être postérieure à la date de début",
		})
	}

	return issues
}

func CreateAnnee(w http.ResponseWriter, r *http.Request) {
	var input CreateAnneeParams
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if issues := validateAnnee(input); len(issues) != 0 {
		services.InvalidRequestError(w, r, "Invalid annee", services.VALIDATION_ERROR, issues)
		return
	}

	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)
	queries := New(pgCtx.Db)

	created, err := queries.CreateAnnee(ctx, input)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, created)
}

func ListAnnee(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)
	queries := New(pgCtx.Db)

	annees, err := queries.ListAnnee(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if annees == nil {
		annees = []Annee{}
	}

	render.JSON(w, r, annees)
}

func GetAnnee(w http.ResponseWriter, r *http.Request) {
	annee := getAnneeFromCtx(r)
	render.JSON(w, r, annee)
}

func UpdateAnnee(w http.ResponseWriter, r *http.Request) {
	oldAnnee := getAnneeFromCtx(r)

	var input UpdateAnneeParams
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if issues := validateAnnee(CreateAnneeParams{Name: input.Name, Debut: input.Debut, Fin: input.Fin}); len(issues) != 0 {
		services.InvalidRequestError(w, r, "Invalid annee", services.VALIDATION_ERROR, issues)
		return
	}

	input.ID = oldAnnee.ID

	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)
	queries := New(pgCtx.Db)

	updated, err := queries.UpdateAnnee(ctx, input)
	if err == pgx.ErrNoRows {
		services.InvalidRequestError(w, r, "Année introuvable", services.NO_INFORMATION, nil)
		return
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	render.JSON(w, r, updated)
}

func DeleteAnnee(w http.ResponseWriter, r *http.Request) {
	annee := getAnneeFromCtx(r)

	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)
	queries := New(pgCtx.Db)

	if err := queries.DeleteAnnee(ctx, annee.ID); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type deleteBulkRequest struct {
	Ids []int64 `json:"ids"`
}

func DeleteAnneeBulk(w http.ResponseWriter, r *http.Request) {
	var input deleteBulkRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)
	queries := New(pgCtx.Db)

	if err := queries.DeleteAnneeByIds(ctx, input.Ids); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
