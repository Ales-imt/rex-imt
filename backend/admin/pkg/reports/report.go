package reports

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"back-rex-common/pkg/services"
	fpdf "github.com/go-pdf/fpdf"
	"github.com/jackc/pgx/v5/pgtype"
)

// ── PDF builder ───────────────────────────────────────────────────────────────

func buildPDF(
	from, to time.Time,
	stats ReportGlobalStatsRow,
	byUrgence []ReportByUrgenceRow,
	byCategorie []ReportByCategorieRow,
	bySentiment []ReportBySentimentRow,
	byPromo []ReportByPromoRow,
) (*fpdf.Fpdf, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// ── header ────────────────────────────────────────────────────────────────
	pdf.SetFillColor(30, 80, 160)
	pdf.Rect(0, 0, 210, 28, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(20, 8)
	pdf.Cell(0, 10, tr("Rapport hebdomadaire \u2014 Feedbacks \u00e9tudiants"))
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(20, 19)
	pdf.Cell(0, 7, tr(fmt.Sprintf("P\u00e9riode : %s \u2014 %s   |   G\u00e9n\u00e9r\u00e9 le %s",
		from.Format("02/01/2006"),
		to.Format("02/01/2006"),
		time.Now().Format("02/01/2006 a 15:04"),
	)))
	pdf.SetTextColor(0, 0, 0)
	pdf.SetY(38)

	// ── section helper ────────────────────────────────────────────────────────
	sectionTitle := func(title string) {
		pdf.Ln(4)
		pdf.SetFillColor(240, 243, 250)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(30, 80, 160)
		pdf.CellFormat(0, 9, "  "+tr(title), "", 1, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(2)
	}

	row2col := func(label, value string) {
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(100, 7, tr(label), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(0, 7, tr(value), "", 1, "L", false, 0, "")
	}

	barRow := func(label string, count, total int64) {
		if total == 0 {
			return
		}
		pct := float64(count) / float64(total) * 100
		barW := (80 * float64(count)) / float64(total)

		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(70, 6, tr(label), "", 0, "L", false, 0, "")
		pdf.SetFillColor(200, 215, 245)
		pdf.CellFormat(barW, 6, "", "", 0, "L", true, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(0, 6, fmt.Sprintf(" %d  (%.0f %%)", count, pct), "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(1)
	}

	// ── vue d'ensemble ────────────────────────────────────────────────────────
	sectionTitle("Vue d'ensemble")
	row2col("Total feedbacks re\u00e7us sur la p\u00e9riode", fmt.Sprintf("%d", stats.Total))
	row2col("Feedbacks analys\u00e9s par l'IA", fmt.Sprintf("%d", stats.Classified))
	row2col("En attente d'analyse", fmt.Sprintf("%d", stats.Pending))

	// ── urgences ──────────────────────────────────────────────────────────────
	sectionTitle("R\u00e9partition par niveau d'urgence")
	urgenceLabel := map[string]string{
		"5": "5 \u2014 Critique (action imm\u00e9diate)",
		"4": "4 \u2014 \u00c9lev\u00e9",
		"3": "3 \u2014 Mod\u00e9r\u00e9",
		"2": "2 \u2014 Faible",
		"1": "1 \u2014 Informatif",
	}
	for _, r := range byUrgence {
		label := urgenceLabel[r.Label]
		if label == "" {
			label = "Urgence " + r.Label
		}
		barRow(label, r.Count, stats.Classified)
	}

	// ── catégories ────────────────────────────────────────────────────────────
	sectionTitle("R\u00e9partition par cat\u00e9gorie")
	for _, r := range byCategorie {
		barRow(r.Label, r.Count, stats.Total)
	}

	// ── sentiments ────────────────────────────────────────────────────────────
	sectionTitle("R\u00e9partition par sentiment")
	sentimentLabel := map[string]string{
		"positif": "Positif",
		"negatif": "N\u00e9gatif",
		"neutre":  "Neutre",
		"mitige":  "Mitig\u00e9",
	}
	for _, r := range bySentiment {
		label := sentimentLabel[r.Label]
		if label == "" {
			label = r.Label
		}
		barRow(label, r.Count, stats.Classified)
	}

	// ── promotions ────────────────────────────────────────────────────────────
	sectionTitle("Activit\u00e9 par promotion")
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(70, 7, "Promotion", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, "Feedbacks", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 7, tr("Urgence max"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 7, "Statut", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for i, r := range byPromo {
		fill := i%2 == 0
		if fill {
			pdf.SetFillColor(248, 250, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		var maxUrgence int64
		switch v := r.MaxUrgence.(type) {
		case int32:
			maxUrgence = int64(v)
		case int64:
			maxUrgence = v
		}
		statut := "OK"
		if maxUrgence >= 5 {
			statut = "Alerte critique"
		} else if maxUrgence >= 3 {
			statut = "Attention"
		}
		pdf.CellFormat(70, 6, tr(r.Promo), "1", 0, "L", fill, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%d", r.Count), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%d", maxUrgence), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(40, 6, statut, "1", 1, "C", fill, 0, "")
	}

	// ── footer ────────────────────────────────────────────────────────────────
	pdf.SetY(-15)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 10, tr(fmt.Sprintf("REX-IMT \u2014 Rapport g\u00e9n\u00e9r\u00e9 automatiquement le %s", time.Now().Format("02/01/2006 a 15:04"))), "", 0, "C", false, 0, "")

	return pdf, pdf.Error()
}

// ── handlers ──────────────────────────────────────────────────────────────────

type periodRequest struct {
	From string `json:"from"` // YYYY-MM-DD
	To   string `json:"to"`   // YYYY-MM-DD
}

func generateAndSend(w http.ResponseWriter, r *http.Request, from, to time.Time, filename string) {
	pgctx := services.GetPgCtx(r.Context())
	ctx := r.Context()
	query := New(pgctx.Db)
	since := pgtype.Timestamptz{Time: from, Valid: true}

	stats, err := query.ReportGlobalStats(ctx, since)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	byUrgence, err := query.ReportByUrgence(ctx, since)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	byCategorie, err := query.ReportByCategorie(ctx, since)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	bySentiment, err := query.ReportBySentiment(ctx, since)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	byPromo, err := query.ReportByPromo(ctx, since)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	pdf, err := buildPDF(from, to, stats, byUrgence, byCategorie, bySentiment, byPromo)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := pdf.Output(w); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
	}
}

func GenerateWeeklyReport(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, 0, -7)
	to := time.Now()
	filename := fmt.Sprintf("rapport-hebdo-%s.pdf", to.Format("2006-01-02"))
	generateAndSend(w, r, from, to, filename)
}

func GeneratePeriodReport(w http.ResponseWriter, r *http.Request) {
	var req periodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		services.InvalidRequestError(w, r, "corps de requ\u00eate invalide", services.VALIDATION_ERROR, nil)
		return
	}
	from, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		services.InvalidRequestError(w, r, "date 'from' invalide, format attendu : YYYY-MM-DD", services.VALIDATION_ERROR, nil)
		return
	}
	to, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		services.InvalidRequestError(w, r, "date 'to' invalide, format attendu : YYYY-MM-DD", services.VALIDATION_ERROR, nil)
		return
	}
	to = to.Add(24*time.Hour - time.Second) // fin de journée

	filename := fmt.Sprintf("rapport-%s_%s.pdf", from.Format("2006-01-02"), to.Format("2006-01-02"))
	generateAndSend(w, r, from, to, filename)
}
