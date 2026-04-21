package reports

import "github.com/go-chi/chi/v5"

func RouteReports(r chi.Router) {
	r.Post("/weekly", GenerateWeeklyReport)
	r.Post("/period", GeneratePeriodReport)
}
