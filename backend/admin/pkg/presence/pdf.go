package presence

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/jackc/pgx/v5/pgtype"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type pdfData struct {
	SeanceID    int64
	Code        string
	MatiereID   int64
	StartsAt    time.Time // instant brut : tri, en-têtes du récapitulatif, bornes
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
	Absents     int // absents NON excusés : Absents et Excuses sont disjoints
	Excuses     int
	Taux        int
	Eleves      []pdfEleveRow
	GeneratedAt time.Time
}

type pdfEleveRow struct {
	Rank     int
	UserID   int32 // clé de rapprochement des lignes du récapitulatif
	Surname  string
	Name     string
	Statut   string
	PointeAt string
	// Justifie : une excuse couvre cette séance. Ce n'est pas un statut — le
	// pointage l'emporte, et le libellé « Excusé » ne remplace « Absent » que
	// lorsque le statut de fait vaut ABSENT.
	Justifie   bool
	HorsGroupe bool
}

// estExcuse : absent ET couvert par une excuse. Seul cas où la ligne change de
// fond et de libellé.
func (e pdfEleveRow) estExcuse() bool {
	return e.Statut != "PRESENT" && e.Statut != "RETARD" && e.Justifie
}

// pdfOptions pilote les sections optionnelles du document. La feuille unitaire
// n'active que Signature ; l'export semestriel expose les quatre au client.
type pdfOptions struct {
	Cover     bool // page de garde + sommaire des matières
	Recap     bool // récapitulatif élèves × séances
	Signature bool // colonne signature vierge dans les feuilles
	Ledger    bool // annexe registre d'intégrité
}

// pdfDoc décrit un document complet : une ou plusieurs feuilles de présence
// partageant une pagination et un pied de page continus.
type pdfDoc struct {
	Promo, Periode string
	Annee          int32
	GeneratedAt    time.Time
	Seances        []pdfData      // triées par matière puis starts_at
	Exclues        []string       // séances non clôturées, pour la page de garde
	Ledger         []pdfLedgerRow // annexe registre, une ligne par séance
	Options        pdfOptions
}

// pdfLedgerRow : empreinte du registre d'intégrité pour une séance. AncreSeq
// vaut 0 lorsqu'aucune ancre TSA ne couvre encore le dernier maillon.
type pdfLedgerRow struct {
	SeanceID   int64
	SeqMin     int64
	SeqMax     int64
	NbMaillons int64
	Hash       string
	AncreSeq   int64
	AncreAt    string
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

// tauxNote énonce la convention de calcul du taux de présence. Elle est
// reproduite en note de bas de page du récapitulatif : cette statistique est
// citée hors de son contexte, sa définition doit voyager avec elle.
const tauxNote = "Taux de présence = (présents + retards) / (effectif inscrit au groupe - excusés). " +
	"Un retard compte comme une présence. Les élèves EXCUSÉS sont retirés du dénominateur : " +
	"une absence excusée ne fait donc pas baisser le taux, elle réduit l'effectif de référence. " +
	"Les élèves hors groupe (H.G.) sont exclus des deux termes et figurent séparément."

// computeTaux applique la convention ci-dessus. Seule implémentation du calcul :
// feuille unitaire, récapitulatif et totaux semestriels s'y réfèrent tous.
//
// Les excusés sortent du dénominateur, sinon un étudiant excusé ferait chuter
// le taux de sa promotion. Un effectif intégralement excusé donne 0 % faute de
// référence — cas signalé par « — » dans le récapitulatif.
func computeTaux(presents, retards, inscrits, excuses int) int {
	reference := inscrits - excuses
	if reference <= 0 {
		return 0
	}
	return ((presents + retards) * 100) / reference
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ─── PDF generation ───────────────────────────────────────────────────────────

// generatePresencePDF génère la feuille d'une séance unique. Ce n'est qu'un cas
// particulier de generateSemestrePDF : un seul chemin de rendu pour les deux.
func generatePresencePDF(w io.Writer, d pdfData) error {
	return generateSemestrePDF(w, pdfDoc{
		Promo:       d.Promo,
		GeneratedAt: d.GeneratedAt,
		Seances:     []pdfData{d},
		Options:     pdfOptions{Signature: true},
	})
}

// generateSemestrePDF assemble le document complet. La pagination et le pied de
// page sont continus sur l'ensemble : le pied lit `cur`, un pointeur et non une
// copie — capturer une pdfData par valeur ferait porter la première séance à
// tout le document.
//
// `cur` est affecté APRÈS AddPage, et non avant : fpdf écrit le pied de la page
// N au moment où l'on ouvre la page N+1 (et celui de la dernière page à la
// fermeture du document). Affecter avant ferait porter à la dernière page de
// chaque séance le code de la séance suivante. Les sauts de page automatiques
// du tableau élèves, eux, tombent en cours de séance et héritent du bon `cur`.
func generateSemestrePDF(w io.Writer, doc pdfDoc) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMarginL, pdfMarginT, pdfMarginR)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AliasNbPages("{nb}")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	var cur *pdfData

	// ── Footer ────────────────────────────────────────────────────────────────
	// La largeur est relue à chaque page : le récapitulatif peut être en paysage.
	pdf.SetFooterFunc(func() {
		pageW, _ := pdf.GetPageSize()
		contentW := pageW - pdfMarginL - pdfMarginR

		gauche := tr(docLabel(doc))
		if cur != nil {
			gauche = tr(fmt.Sprintf("Séance #%d · Code %s · Registre d'intégrité cryptographique", cur.SeanceID, cur.Code))
		}

		pdf.SetY(-13)
		pdf.SetDrawColor(200, 200, 215)
		pdf.SetLineWidth(0.2)
		pdf.Line(pdfMarginL, pdf.GetY(), pdfMarginL+contentW, pdf.GetY())
		pdf.Ln(1)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(150, 150, 150)
		pdf.CellFormat(contentW*2/3, 4, gauche, "", 0, "L", false, 0, "")
		pdf.CellFormat(contentW/3, 4,
			tr(fmt.Sprintf("Page %d/{nb} · Généré le %s", pdf.PageNo(), doc.GeneratedAt.Format("02/01/2006 à 15:04"))),
			"", 0, "R", false, 0, "")
	})

	blocs := blocsMatiere(doc.Seances)

	if doc.Options.Cover {
		pdf.AddPage()
		renderCover(pdf, tr, doc, computeTotaux(doc))
		pdf.AddPage()
		renderSommaire(pdf, tr, blocs)
	}

	if doc.Options.Recap && len(doc.Seances) > 0 {
		renderRecap(pdf, tr, doc)
	}

	// Index des séances ouvrant une matière : le sommaire y renvoie par jeton,
	// résolu ici puisque le numéro de page n'est connu qu'à la composition.
	ouvre := make(map[int]int, len(blocs))
	for i, b := range blocs {
		ouvre[b.Debut] = i
	}

	for i := range doc.Seances {
		pdf.AddPage()
		cur = &doc.Seances[i]
		if j, ok := ouvre[i]; ok {
			pdf.RegisterAlias(aliasMatiere(j), fmt.Sprintf("%-4d", pdf.PageNo()))
		}
		renderSeanceSheet(pdf, tr, doc.Seances[i], doc.Options)
	}

	if doc.Options.Ledger && len(doc.Seances) > 0 {
		cur = nil
		renderLedgerAnnexe(pdf, tr, doc)
	}

	// Un document sans aucune page n'est pas un PDF valide : à zéro séance et
	// sans page de garde, on émet une page expliquant l'absence de résultat.
	if pdf.PageNo() == 0 {
		pdf.AddPage()
		cur = nil
		renderDocumentVide(pdf, tr, doc)
	}

	return pdf.Output(w)
}

// docLabel : identification du document en pied des pages hors séance.
func docLabel(doc pdfDoc) string {
	parts := make([]string, 0, 3)
	if doc.Promo != "" {
		parts = append(parts, doc.Promo)
	}
	if doc.Periode != "" {
		parts = append(parts, doc.Periode)
	}
	if doc.Annee > 0 {
		parts = append(parts, fmt.Sprintf("%d-%d", doc.Annee, doc.Annee+1))
	}
	if len(parts) == 0 {
		return "Feuilles de présence"
	}
	return strings.Join(parts, " · ")
}

func renderDocumentVide(pdf *fpdf.Fpdf, tr func(string) string, doc pdfDoc) {
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(50, 45, 115)
	pdf.SetXY(pdfMarginL, pdfMarginT+40)
	pdf.CellFormat(pdfContentW, 8, tr("Aucune séance à éditer"), "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(90, 90, 105)
	pdf.CellFormat(pdfContentW, 6, tr(docLabel(doc)), "", 1, "C", false, 0, "")
}

// renderSeanceSheet dessine la feuille d'une séance sur la page courante. Ne
// crée jamais la première page (l'appelant s'en charge) ; les pages de
// débordement du tableau élèves restent de sa responsabilité.
func renderSeanceSheet(pdf *fpdf.Fpdf, tr func(string) string, d pdfData, opts pdfOptions) {
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
	statsW := pdfContentW / 5

	type statBox struct {
		label   string
		value   string
		r, g, b int
	}
	stats := [5]statBox{
		{"Présents", fmt.Sprintf("%d / %d", d.Presents, d.Total), 22, 163, 74},
		{"Retards", fmt.Sprintf("%d", d.Retards), 200, 115, 10},
		{"Absents", fmt.Sprintf("%d", d.Absents), 200, 35, 35},
		{"Excusés", fmt.Sprintf("%d", d.Excuses), 146, 90, 10},
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

	// Colonnes : # | Nom | Prénom | Statut | Heure [| Signature]
	// Sans colonne signature, ses 34mm reviennent aux noms (2×17mm) : le total
	// reste pdfContentW, les bordures restent alignées sur la grille.
	colWidths := []float64{8, 42, 42, 30, 24, 34}
	headers := []string{"#", tr("NOM"), tr("PRÉNOM"), "STATUT", "HEURE", "SIGNATURE"}
	if !opts.Signature {
		colWidths = []float64{8, 59, 59, 30, 24}
		headers = headers[:5]
	}

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
		// Le fond jaune pâle remplace le zébrage : c'est la LIGNE qui porte
		// l'excuse. Le retard, lui, reste un simple libellé orange sur fond
		// zébré normal — les deux registres ne se rencontrent jamais, un
		// excusé étant par définition absent.
		switch {
		case e.estExcuse():
			pdf.SetFillColor(255, 247, 214)
		case i%2 == 0:
			pdf.SetFillColor(251, 250, 255)
		default:
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
		switch {
		case e.Statut == "PRESENT":
			sr, sg, sb = 22, 163, 74
			statutLabel = "Présent"
		case e.Statut == "RETARD":
			sr, sg, sb = 200, 115, 10
			statutLabel = "Retard"
		case e.Justifie:
			sr, sg, sb = 146, 90, 10
			statutLabel = "Excusé"
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
		for j := 4; j < len(colWidths); j++ {
			txt := ""
			if j == 4 {
				txt = "  " + tr(e.PointeAt)
			}
			pdf.CellFormat(colWidths[j], 7, txt, "B", 0, "L", true, 0, "")
		}

		pdf.Ln(-1)
	}
}

// ─── Handler data builder ─────────────────────────────────────────────────────

// seanceInfo est la forme commune aux deux origines d'une séance :
// GetSeanceDetail (feuille unitaire) et ListSeancesByPeriode (export
// semestriel). Elle évite de dupliquer buildPdfData pour deux lignes sqlc qui
// ne diffèrent que par leur type généré.
type seanceInfo struct {
	ID               int64
	MatiereID        int64
	Code             string
	MatiereName      string
	Promo            string
	Prof             string
	Salle            string
	StartsAt         pgtype.Timestamptz
	EndsAt           pgtype.Timestamptz
	OpenedAt         pgtype.Timestamptz
	ClosedAt         pgtype.Timestamptz
	LateAfterMinutes int32
}

func seanceInfoFromDetail(s GetSeanceDetailRow) seanceInfo {
	return seanceInfo{
		ID: s.ID, Code: s.Code.String, MatiereName: s.MatiereName, Promo: s.Promo,
		Prof: s.Prof, Salle: s.Salle,
		StartsAt: s.StartsAt, EndsAt: s.EndsAt, OpenedAt: s.OpenedAt, ClosedAt: s.ClosedAt,
		LateAfterMinutes: s.LateAfterMinutes,
	}
}

func seanceInfoFromPeriode(s ListSeancesByPeriodeRow) seanceInfo {
	return seanceInfo{
		ID: s.ID, MatiereID: s.MatiereID, Code: s.Code.String,
		MatiereName: s.MatiereName, Promo: s.Promo, Prof: s.Prof, Salle: s.Salle,
		StartsAt: s.StartsAt, EndsAt: s.EndsAt, OpenedAt: s.OpenedAt, ClosedAt: s.ClosedAt,
		LateAfterMinutes: s.LateAfterMinutes,
	}
}

// presenceLigne : ligne de présence indépendante de la requête qui l'a produite.
type presenceLigne struct {
	UserID   int32
	Name     string
	Surname  string
	Statut   string
	PointeAt pgtype.Timestamptz
	Justifie bool
}

// buildPdfData assembles pdfData from DB rows.
//
// Convention de taux (cf. tauxNote) : le dénominateur est l'effectif inscrit au
// groupe de la séance, le numérateur les présents ET les retards. Les élèves
// hors groupe sont exclus des deux termes — ils étaient auparavant comptés au
// seul numérateur, ce qui pouvait produire un taux supérieur à 100 %.
func buildPdfData(seance seanceInfo, rows, horsGroupe []presenceLigne, generatedAt time.Time) pdfData {
	var presents, retards, absents, excuses int
	eleves := make([]pdfEleveRow, 0, len(rows)+len(horsGroupe))
	rank := 1
	for _, r := range rows {
		switch {
		case r.Statut == "PRESENT":
			presents++
		case r.Statut == "RETARD":
			retards++
		case r.Justifie:
			excuses++
		default:
			absents++
		}
		var pointeAt string
		if r.PointeAt.Valid {
			pointeAt = formatHeure(r.PointeAt.Time)
		}
		eleves = append(eleves, pdfEleveRow{
			Rank:     rank,
			UserID:   r.UserID,
			Surname:  capitalize(r.Surname),
			Name:     capitalize(r.Name),
			Statut:   r.Statut,
			PointeAt: pointeAt,
			Justifie: r.Justifie,
		})
		rank++
	}
	// Les hors-groupe sont listés sur la feuille mais restent hors des compteurs :
	// ils ne sont inscrits à aucun des deux termes du taux.
	for _, r := range horsGroupe {
		var pointeAt string
		if r.PointeAt.Valid {
			pointeAt = formatHeure(r.PointeAt.Time)
		}
		eleves = append(eleves, pdfEleveRow{
			Rank:       rank,
			UserID:     r.UserID,
			Surname:    capitalize(r.Surname),
			Name:       capitalize(r.Name),
			Statut:     r.Statut,
			PointeAt:   pointeAt,
			Justifie:   r.Justifie,
			HorsGroupe: true,
		})
		rank++
	}

	total := len(rows)
	taux := computeTaux(presents, retards, total, excuses)

	// Date + horaire
	var dateStr, horaireStr string
	var startsAt time.Time
	if seance.StartsAt.Valid {
		startsAt = seance.StartsAt.Time
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
		Code:        seance.Code,
		MatiereID:   seance.MatiereID,
		StartsAt:    startsAt,
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
		Excuses:     excuses,
		Taux:        taux,
		Eleves:      eleves,
		GeneratedAt: generatedAt,
	}
}
