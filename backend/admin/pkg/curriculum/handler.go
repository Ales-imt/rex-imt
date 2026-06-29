package curriculum

import (
	"back-rex-common/pkg/services"
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
)

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

func GetAnnees(w http.ResponseWriter, r *http.Request) {
	q := New(services.GetPgCtx(r.Context()).Db)
	annees, err := q.GetDistinctAnnees(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if annees == nil {
		annees = []int32{}
	}
	render.JSON(w, r, annees)
}

func GetPromotionTree(w http.ResponseWriter, r *http.Request) {
	anneeStr := r.URL.Query().Get("annee")
	var annee int32
	if anneeStr != "" {
		v, err := strconv.ParseInt(anneeStr, 10, 32)
		if err != nil {
			services.InvalidRequestError(w, r, "annee invalide", services.NO_INFORMATION, nil)
			return
		}
		annee = int32(v)
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	if annee == 0 {
		annees, err := q.GetDistinctAnnees(ctx)
		if err != nil || len(annees) == 0 {
			render.JSON(w, r, []promotionTree{})
			return
		}
		annee = annees[0]
	}

	rows, err := q.GetAllPromotionTree(ctx, annee)
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
