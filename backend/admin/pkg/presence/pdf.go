package presence

import (
	presencedata "back-rex-common/pkg/presencedata/gen"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type pdfData struct {
	SeanceID    int64
	Code        string
	Matiere     string
	Promo       string
	Prof        string
	Salle       string
	Date        string
	Horaire     string
	OpenedAt    string
	ClosedAt    string
	LateMin     int32
	Total       int
	Presents    int
	Retards     int
	Absents     int
	Taux        int
	Eleves      []pdfEleveRow
	GeneratedAt time.Time
}

type pdfEleveRow struct {
	Rank       int
	Surname    string
	Name       string
	Statut     string
	PointeAt   string
	HorsGroupe bool
}

// ─── Layout constants ─────────────────────────────────────────────────────────

const (
	pdfMarginL  = 15.0
	pdfMarginT  = 15.0
	pdfMarginR  = 15.0
	pdfContentW = 210.0 - pdfMarginL - pdfMarginR // 180mm

	headerH  = 22.0
	labelH   = 4.5
	valueH   = 7.0
	infoRowH = labelH + valueH // 11.5mm per info row
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

var frDays = [7]string{"Dimanche", "Lundi", "Mardi", "Mercredi", "Jeudi", "Vendredi", "Samedi"}
var frMonths = [13]string{"", "janvier", "février", "mars", "avril", "mai", "juin",
	"juillet", "août", "septembre", "octobre", "novembre", "décembre"}

// parisLoc : les séances et pointages sont des instants ; on les affiche en
// heure de Paris (fuseau des plannings). tzdata est embarqué dans le binaire
// (cf. cmd/main.go), donc LoadLocation aboutit ; UTC en ultime filet.
var parisLoc = func() *time.Location {
	l, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.UTC
	}
	return l
}()

func formatDateFR(t time.Time) string {
	t = t.In(parisLoc)
	return fmt.Sprintf("%s %d %s %d", frDays[t.Weekday()], t.Day(), frMonths[t.Month()], t.Year())
}

func formatHeure(t time.Time) string {
	return t.In(parisLoc).Format("15:04")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ─── PDF generation ───────────────────────────────────────────────────────────

func generatePresencePDF(w io.Writer, d pdfData) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMarginL, pdfMarginT, pdfMarginR)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AliasNbPages("{nb}")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// ── Footer ────────────────────────────────────────────────────────────────
	pdf.SetFooterFunc(func() {
		pdf.SetY(-13)
		pdf.SetDrawColor(200, 200, 215)
		pdf.SetLineWidth(0.2)
		pdf.Line(pdfMarginL, pdf.GetY(), pdfMarginL+pdfContentW, pdf.GetY())
		pdf.Ln(1)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(150, 150, 150)
		pdf.CellFormat(pdfContentW*2/3, 4,
			tr(fmt.Sprintf("Séance #%d · Code %s · Registre d'intégrité cryptographique", d.SeanceID, d.Code)),
			"", 0, "L", false, 0, "")
		pdf.CellFormat(pdfContentW/3, 4,
			tr(fmt.Sprintf("Page %d/{nb} · Généré le %s", pdf.PageNo(), d.GeneratedAt.Format("02/01/2006 à 15:04"))),
			"", 0, "R", false, 0, "")
	})

	pdf.AddPage()

	// ── Header band ───────────────────────────────────────────────────────────
	pdf.SetFillColor(50, 45, 115)
	pdf.Rect(pdfMarginL, pdfMarginT, pdfContentW, headerH, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(pdfMarginL+5, pdfMarginT+3.5)
	pdf.CellFormat(110, 8, tr("FEUILLE DE PRÉSENCE"), "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(pdfMarginL+5, pdfMarginT+12.5)
	pdf.CellFormat(110, 5, "IMT Mines "+tr("Alès"), "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(pdfMarginL+pdfContentW-62, pdfMarginT+3.5)
	pdf.CellFormat(57, 8, tr(fmt.Sprintf("Séance  #%d", d.SeanceID)), "", 0, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(pdfMarginL+pdfContentW-62, pdfMarginT+12.5)
	pdf.CellFormat(57, 5, tr(fmt.Sprintf("Code : %s", d.Code)), "", 0, "R", false, 0, "")

	pdf.SetTextColor(0, 0, 0)

	// ── Info grid (3 cols × 3 rows) ───────────────────────────────────────────
	colW := pdfContentW / 3
	info := [9][2]string{
		{tr("Matière"), tr(d.Matiere)},
		{"Promotion", tr(d.Promo)},
		{tr("Enseignant"), tr(d.Prof)},
		{"Date", tr(d.Date)},
		{"Horaire", tr(d.Horaire)},
		{"Salle", tr(d.Salle)},
		{"Ouverture", tr(d.OpenedAt)},
		{tr("Clôture"), tr(d.ClosedAt)},
		{tr("Retard après"), tr(fmt.Sprintf("%d min", d.LateMin))},
	}

	pdf.SetLineWidth(0.15)
	pdf.SetDrawColor(205, 200, 225)

	gridY := pdfMarginT + headerH + 4
	for i, field := range info {
		col := i % 3
		row := i / 3
		x := pdfMarginL + float64(col)*colW
		cellY := gridY + float64(row)*infoRowH

		pdf.SetFillColor(242, 240, 252)
		pdf.SetTextColor(80, 65, 155)
		pdf.SetFont("Helvetica", "B", 7)
		pdf.SetXY(x, cellY)
		pdf.CellFormat(colW, labelH, "  "+field[0], "1", 0, "L", true, 0, "")

		pdf.SetFillColor(255, 255, 255)
		pdf.SetTextColor(20, 20, 20)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(x, cellY+labelH)
		pdf.CellFormat(colW, valueH, "  "+field[1], "1", 0, "L", true, 0, "")
	}

	// ── Stats bar ─────────────────────────────────────────────────────────────
	statsY := gridY + 3*infoRowH + 5
	statsW := pdfContentW / 4

	type statBox struct {
		label string
		value string
		r, g, b int
	}
	stats := [4]statBox{
		{tr("Présents"), fmt.Sprintf("%d / %d", d.Presents, d.Total), 22, 163, 74},
		{"Retards", fmt.Sprintf("%d", d.Retards), 200, 115, 10},
		{"Absents", fmt.Sprintf("%d", d.Absents), 200, 35, 35},
		{"Taux", fmt.Sprintf("%d%%", d.Taux), 50, 45, 115},
	}
	for i, s := range stats {
		x := pdfMarginL + float64(i)*statsW
		pdf.SetFillColor(s.r, s.g, s.b)
		pdf.Rect(x, statsY, statsW, 16, "F")
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 15)
		pdf.SetXY(x, statsY+1.5)
		pdf.CellFormat(statsW, 8, s.value, "", 0, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetXY(x, statsY+10)
		pdf.CellFormat(statsW, 5, tr(s.label), "", 0, "C", false, 0, "")
	}

	// ── Student table ─────────────────────────────────────────────────────────
	tableY := statsY + 21
	pdf.SetXY(pdfMarginL, tableY)

	// Column widths: # | Nom | Prénom | Statut | Heure | Signature
	colWidths := [6]float64{8, 42, 42, 30, 24, 34}
	headers := [6]string{"#", tr("NOM"), tr("PRÉNOM"), "STATUT", "HEURE", "SIGNATURE"}

	// Header row
	pdf.SetFillColor(50, 45, 115)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetLineWidth(0)
	for j, h := range headers {
		align := "L"
		if j == 0 {
			align = "C"
		}
		pdf.CellFormat(colWidths[j], 7, "  "+h, "", 0, align, true, 0, "")
	}
	pdf.Ln(-1)

	// Body rows
	pdf.SetLineWidth(0.1)
	pdf.SetDrawColor(215, 215, 228)

	for i, e := range d.Eleves {
		if pdf.GetY() > 265 {
			pdf.AddPage()
			// Repeat table header on new page
			pdf.SetFillColor(50, 45, 115)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Helvetica", "B", 8)
			pdf.SetLineWidth(0)
			for j, h := range headers {
				align := "L"
				if j == 0 {
					align = "C"
				}
				pdf.CellFormat(colWidths[j], 7, "  "+h, "", 0, align, true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetLineWidth(0.1)
		}

		rowY := pdf.GetY()
		if i%2 == 0 {
			pdf.SetFillColor(251, 250, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.SetXY(pdfMarginL, rowY)
		pdf.SetTextColor(80, 80, 95)
		pdf.SetFont("Helvetica", "", 8)
		pdf.CellFormat(colWidths[0], 7, fmt.Sprintf("%d", e.Rank), "B", 0, "C", true, 0, "")

		pdf.SetTextColor(20, 20, 20)
		pdf.CellFormat(colWidths[1], 7, "  "+tr(e.Surname), "B", 0, "L", true, 0, "")
		pdf.CellFormat(colWidths[2], 7, "  "+tr(e.Name), "B", 0, "L", true, 0, "")

		// Statut with color
		var sr, sg, sb int
		var statutLabel string
		switch e.Statut {
		case "PRESENT":
			sr, sg, sb = 22, 163, 74
			statutLabel = "Présent"
		case "RETARD":
			sr, sg, sb = 200, 115, 10
			statutLabel = "Retard"
		default:
			sr, sg, sb = 200, 35, 35
			statutLabel = "Absent"
		}
		if e.HorsGroupe {
			statutLabel += " (H.G.)"
		}
		pdf.SetTextColor(sr, sg, sb)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(colWidths[3], 7, "  "+tr(statutLabel), "B", 0, "L", true, 0, "")

		pdf.SetTextColor(70, 70, 85)
		pdf.SetFont("Helvetica", "", 8)
		pdf.CellFormat(colWidths[4], 7, "  "+tr(e.PointeAt), "B", 0, "L", true, 0, "")
		pdf.CellFormat(colWidths[5], 7, "", "B", 0, "L", true, 0, "")

		pdf.Ln(-1)
	}

	return pdf.Output(w)
}

// ─── Handler data builder ─────────────────────────────────────────────────────

// buildPdfData assembles pdfData from DB rows.
func buildPdfData(seance GetSeanceDetailRow, rows []presencedata.ListPresenceRow, horsGroupe []presencedata.ListPresenceHorsGroupeRow) pdfData {
	var presents, retards, absents int
	eleves := make([]pdfEleveRow, 0, len(rows)+len(horsGroupe))
	rank := 1
	for _, r := range rows {
		switch r.Statut {
		case "PRESENT":
			presents++
		case "RETARD":
			retards++
		default:
			absents++
		}
		var pointeAt string
		if r.PointeAt.Valid {
			pointeAt = formatHeure(r.PointeAt.Time)
		}
		eleves = append(eleves, pdfEleveRow{
			Rank:     rank,
			Surname:  capitalize(r.Surname),
			Name:     capitalize(r.Name),
			Statut:   r.Statut,
			PointeAt: pointeAt,
		})
		rank++
	}
	for _, r := range horsGroupe {
		switch r.Statut {
		case "PRESENT":
			presents++
		case "RETARD":
			retards++
		default:
			absents++
		}
		var pointeAt string
		if r.PointeAt.Valid {
			pointeAt = formatHeure(r.PointeAt.Time)
		}
		eleves = append(eleves, pdfEleveRow{
			Rank:       rank,
			Surname:    capitalize(r.Surname),
			Name:       capitalize(r.Name),
			Statut:     r.Statut,
			PointeAt:   pointeAt,
			HorsGroupe: true,
		})
		rank++
	}

	total := len(rows)
	taux := 0
	if total > 0 {
		taux = (presents * 100) / total
	}

	// Date + horaire
	var dateStr, horaireStr string
	if seance.StartsAt.Valid {
		dateStr = formatDateFR(seance.StartsAt.Time)
		horaireStr = formatHeure(seance.StartsAt.Time)
		if seance.EndsAt.Valid {
			horaireStr += " – " + formatHeure(seance.EndsAt.Time)
		}
	}
	var openedStr, closedStr string
	if seance.OpenedAt.Valid {
		openedStr = formatHeure(seance.OpenedAt.Time)
	}
	if seance.ClosedAt.Valid {
		closedStr = formatHeure(seance.ClosedAt.Time)
	}

	return pdfData{
		SeanceID:    seance.ID,
		Code:        seance.Code.String,
		Matiere:     seance.MatiereName,
		Promo:       seance.Promo,
		Prof:        seance.Prof,
		Salle:       seance.Salle,
		Date:        dateStr,
		Horaire:     horaireStr,
		OpenedAt:    openedStr,
		ClosedAt:    closedStr,
		LateMin:     seance.LateAfterMinutes,
		Total:       total,
		Presents:    presents,
		Retards:     retards,
		Absents:     absents,
		Taux:        taux,
		Eleves:      eleves,
		GeneratedAt: time.Now().UTC(),
	}
}
