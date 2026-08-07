package presence

// Sections du document semestriel : page de garde, sommaire, récapitulatif
// élèves × séances et annexe registre d'intégrité. Les feuilles de séance
// elles-mêmes restent dans pdf.go.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// ─── Constantes de mise en page ───────────────────────────────────────────────

const (
	// Au-delà de ce nombre de séances, le récapitulatif bascule en paysage :
	// en portrait les colonnes deviendraient trop étroites pour rester lisibles.
	recapSeuilPaysage = 12

	// Nombre maximum de séances exclues nommées en page de garde. Sur un
	// semestre où rien n'a été clôturé, les nommer toutes produirait des
	// dizaines de pages de puces avant le sommaire ; le total, lui, est
	// toujours annoncé.
	maxExcluesNommees = 40

	recapColW  = 8.0  // largeur d'une colonne séance
	recapNomW  = 52.0 // colonne « élève »
	recapRowH  = 5.0
	recapTauxW = 14.0 // colonne de taux individuel, en fin de bloc

	// Largeur utile d'une page A4 paysage.
	pdfContentWPaysage = 297.0 - pdfMarginL - pdfMarginR
)

// formatA4 est toujours donné en portrait, même pour une page paysage :
// AddPageFormat échange lui-même largeur et hauteur quand l'orientation est
// « L ». Lui passer {297, 210} produirait une page portrait.
var formatA4 = fpdf.SizeType{Wd: 210, Ht: 297}

// aliasMatiere : jeton substitué par le numéro de page réel de la matière une
// fois celle-ci atteinte (fpdf.RegisterAlias). Longueur fixe pour que le
// remplacement ne déplace pas le texte de la ligne du sommaire.
func aliasMatiere(i int) string { return fmt.Sprintf("{{p%03d}}", i) }

// ─── Totaux ───────────────────────────────────────────────────────────────────

type pdfTotaux struct {
	NbMatieres int
	NbSeances  int
	Presents   int
	Retards    int
	Inscrits   int // somme des effectifs séance par séance
	Taux       int
	Du, Au     time.Time
}

func computeTotaux(doc pdfDoc) pdfTotaux {
	t := pdfTotaux{NbSeances: len(doc.Seances)}
	matieres := make(map[int64]struct{}, len(doc.Seances))
	for _, s := range doc.Seances {
		matieres[s.MatiereID] = struct{}{}
		t.Presents += s.Presents
		t.Retards += s.Retards
		t.Inscrits += s.Total

		// Les séances sont triées par matière puis date : les bornes
		// temporelles demandent un balayage, pas les extrémités de la tranche.
		if s.StartsAt.IsZero() {
			continue
		}
		if t.Du.IsZero() || s.StartsAt.Before(t.Du) {
			t.Du = s.StartsAt
		}
		if t.Au.IsZero() || s.StartsAt.After(t.Au) {
			t.Au = s.StartsAt
		}
	}
	t.NbMatieres = len(matieres)
	t.Taux = computeTaux(t.Presents, t.Retards, t.Inscrits)
	return t
}

// ─── Page de garde ────────────────────────────────────────────────────────────

func renderCover(pdf *fpdf.Fpdf, tr func(string) string, doc pdfDoc, t pdfTotaux) {
	// Bandeau
	pdf.SetFillColor(50, 45, 115)
	pdf.Rect(pdfMarginL, pdfMarginT, pdfContentW, 34, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetXY(pdfMarginL+6, pdfMarginT+6)
	pdf.CellFormat(pdfContentW-12, 10, tr("FEUILLES DE PRÉSENCE"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetX(pdfMarginL + 6)
	pdf.CellFormat(pdfContentW-12, 7, tr(docLabel(doc)), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetX(pdfMarginL + 6)
	pdf.CellFormat(pdfContentW-12, 5, "IMT Mines "+tr("Alès"), "", 1, "L", false, 0, "")

	pdf.SetTextColor(0, 0, 0)
	y := pdfMarginT + 34 + 10

	// Identification
	periode := "—"
	if !t.Du.IsZero() {
		periode = fmt.Sprintf("%s au %s",
			t.Du.In(parisLoc).Format("02/01/2006"), t.Au.In(parisLoc).Format("02/01/2006"))
	}
	// Libellés et valeurs bruts : tr est appliqué une seule fois, au rendu.
	lignes := [][2]string{
		{"Promotion", doc.Promo},
		{"Semestre", doc.Periode},
		{"Année universitaire", fmt.Sprintf("%d-%d", doc.Annee, doc.Annee+1)},
		{"Période couverte", periode},
		{"Généré le", doc.GeneratedAt.In(parisLoc).Format("02/01/2006 à 15:04")},
	}
	for _, l := range lignes {
		pdf.SetXY(pdfMarginL, y)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(80, 65, 155)
		pdf.CellFormat(55, 7, tr(l[0]), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(20, 20, 20)
		pdf.CellFormat(pdfContentW-55, 7, tr(l[1]), "", 1, "L", false, 0, "")
		y += 7
	}

	// Totaux
	y += 8
	stats := [4][2]string{
		{fmt.Sprintf("%d", t.NbMatieres), tr("Matières")},
		{fmt.Sprintf("%d", t.NbSeances), tr("Séances")},
		{fmt.Sprintf("%d", t.Presents+t.Retards), tr("Présences relevées")},
		{fmt.Sprintf("%d%%", t.Taux), "Taux moyen"},
	}
	boxW := pdfContentW / 4
	for i, s := range stats {
		x := pdfMarginL + float64(i)*boxW
		pdf.SetFillColor(242, 240, 252)
		pdf.Rect(x, y, boxW-2, 18, "F")
		pdf.SetFont("Helvetica", "B", 15)
		pdf.SetTextColor(50, 45, 115)
		pdf.SetXY(x, y+2)
		pdf.CellFormat(boxW-2, 8, s[0], "", 0, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(90, 90, 105)
		pdf.SetXY(x, y+11)
		pdf.CellFormat(boxW-2, 4, s[1], "", 0, "C", false, 0, "")
	}
	y += 26

	pdf.SetXY(pdfMarginL, y)
	pdf.SetFont("Helvetica", "I", 7)
	pdf.SetTextColor(120, 120, 135)
	pdf.MultiCell(pdfContentW, 3.5, tr(tauxNote), "", "L", false)
	y = pdf.GetY() + 8

	// Séances exclues — nommées, pour que l'absence soit vérifiable.
	pdf.SetXY(pdfMarginL, y)
	if len(doc.Exclues) == 0 {
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(22, 120, 60)
		pdf.CellFormat(pdfContentW, 6,
			tr("Toutes les séances de la période sélectionnée sont clôturées et figurent dans ce document."),
			"", 1, "L", false, 0, "")
		return
	}

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(180, 95, 10)
	pdf.CellFormat(pdfContentW, 6,
		tr(fmt.Sprintf("%d séance(s) non clôturée(s), exclues de ce document", len(doc.Exclues))),
		"", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(70, 70, 85)
	pdf.SetX(pdfMarginL)
	pdf.MultiCell(pdfContentW, 4,
		tr("Leur pointage est encore ouvert : les relevés ne sont pas définitifs. "+
			"Clôturez-les puis relancez l'export pour les y intégrer."),
		"", "L", false)
	pdf.Ln(2)
	nommees := min(len(doc.Exclues), maxExcluesNommees)
	for _, e := range doc.Exclues[:nommees] {
		pdf.SetX(pdfMarginL + 4)
		pdf.SetTextColor(70, 70, 85)
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(pdfContentW-4, 4.5, tr("• "+e), "", "L", false)
	}
	if reste := len(doc.Exclues) - nommees; reste > 0 {
		pdf.SetX(pdfMarginL + 4)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(120, 120, 135)
		pdf.MultiCell(pdfContentW-4, 4.5,
			tr(fmt.Sprintf("… et %d autre(s) séance(s) non clôturée(s), non détaillées ici.", reste)),
			"", "L", false)
	}
}

// ─── Sommaire ─────────────────────────────────────────────────────────────────

// matiereBloc : suite de séances consécutives d'une même matière. Les séances
// étant triées par matière puis date, un simple balayage suffit à les délimiter.
type matiereBloc struct {
	MatiereID int64
	Nom       string
	Debut     int // index de la première séance dans doc.Seances
	Nb        int
}

func blocsMatiere(seances []pdfData) []matiereBloc {
	var blocs []matiereBloc
	for i, s := range seances {
		if n := len(blocs); n > 0 && blocs[n-1].MatiereID == s.MatiereID {
			blocs[n-1].Nb++
			continue
		}
		blocs = append(blocs, matiereBloc{MatiereID: s.MatiereID, Nom: s.Matiere, Debut: i, Nb: 1})
	}
	return blocs
}

// renderSommaire écrit les numéros de page sous forme de jetons, remplacés par
// leur valeur réelle quand la matière est atteinte : les pages n'étant pas
// encore composées, leur numéro est inconnu à cet instant.
func renderSommaire(pdf *fpdf.Fpdf, tr func(string) string, blocs []matiereBloc) {
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(50, 45, 115)
	pdf.CellFormat(pdfContentW, 9, tr("Sommaire des matières"), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	if len(blocs) == 0 {
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(90, 90, 105)
		pdf.CellFormat(pdfContentW, 6,
			tr("Aucune feuille de présence dans ce document (voir la page de garde)."),
			"", 1, "L", false, 0, "")
		return
	}

	pdf.SetFont("Helvetica", "B", 7)
	pdf.SetFillColor(50, 45, 115)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(pdfContentW-46, 6, "  "+tr("MATIÈRE"), "", 0, "L", true, 0, "")
	pdf.CellFormat(24, 6, tr("SÉANCES"), "", 0, "C", true, 0, "")
	pdf.CellFormat(22, 6, "PAGE", "", 1, "C", true, 0, "")

	pdf.SetDrawColor(215, 215, 228)
	pdf.SetLineWidth(0.1)
	for i, b := range blocs {
		if i%2 == 0 {
			pdf.SetFillColor(251, 250, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(20, 20, 20)
		pdf.CellFormat(pdfContentW-46, 6, "  "+tr(b.Nom), "B", 0, "L", true, 0, "")
		pdf.SetTextColor(70, 70, 85)
		pdf.CellFormat(24, 6, fmt.Sprintf("%d", b.Nb), "B", 0, "C", true, 0, "")
		// Cellule alignée à gauche : le jeton et son remplacement n'ont pas la
		// même longueur, un alignement à droite décalerait le nombre.
		pdf.CellFormat(22, 6, "  "+aliasMatiere(i), "B", 1, "L", true, 0, "")
	}
}

// ─── Récapitulatif élèves × séances ───────────────────────────────────────────

type recapEleve struct {
	UserID  int32
	Surname string
	Name    string
}

// recapModele agrège les feuilles en une matrice élève × séance.
func recapModele(seances []pdfData) ([]recapEleve, map[int32]map[int64]string) {
	parEleve := make(map[int32]recapEleve)
	statuts := make(map[int32]map[int64]string)

	for _, s := range seances {
		for _, e := range s.Eleves {
			// Les hors-groupe ne sont inscrits à aucune séance du semestre :
			// une ligne pour eux serait vide partout sauf en un point.
			if e.HorsGroupe {
				continue
			}
			parEleve[e.UserID] = recapEleve{UserID: e.UserID, Surname: e.Surname, Name: e.Name}
			if statuts[e.UserID] == nil {
				statuts[e.UserID] = make(map[int64]string, len(seances))
			}
			statuts[e.UserID][s.SeanceID] = e.Statut
		}
	}

	eleves := make([]recapEleve, 0, len(parEleve))
	for _, e := range parEleve {
		eleves = append(eleves, e)
	}
	sort.Slice(eleves, func(i, j int) bool {
		if eleves[i].Surname != eleves[j].Surname {
			return eleves[i].Surname < eleves[j].Surname
		}
		if eleves[i].Name != eleves[j].Name {
			return eleves[i].Name < eleves[j].Name
		}
		return eleves[i].UserID < eleves[j].UserID
	})
	return eleves, statuts
}

func renderRecap(pdf *fpdf.Fpdf, tr func(string) string, doc pdfDoc) {
	eleves, statuts := recapModele(doc.Seances)

	paysage := len(doc.Seances) > recapSeuilPaysage
	contentW := pdfContentW
	if paysage {
		contentW = pdfContentWPaysage
	}
	// Nombre de colonnes séance tenant sur une page ; au-delà, le récapitulatif
	// est découpé en blocs de colonnes, chacun reprenant la colonne des noms.
	parPage := int((contentW - recapNomW - recapTauxW) / recapColW)
	if parPage < 1 {
		parPage = 1
	}

	for debut := 0; debut < len(doc.Seances); debut += parPage {
		fin := min(debut+parPage, len(doc.Seances))

		if paysage {
			pdf.AddPageFormat("L", formatA4)
		} else {
			pdf.AddPage()
		}
		renderRecapBloc(pdf, tr, doc, doc.Seances[debut:fin], debut, eleves, statuts, contentW)
	}
}

func renderRecapBloc(
	pdf *fpdf.Fpdf, tr func(string) string, doc pdfDoc,
	bloc []pdfData, offset int, eleves []recapEleve,
	statuts map[int32]map[int64]string, contentW float64,
) {
	titre := tr("Récapitulatif de présence")
	if offset > 0 || len(bloc) < len(doc.Seances) {
		titre += tr(fmt.Sprintf(" — séances %d à %d", offset+1, offset+len(bloc)))
	}
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(50, 45, 115)
	pdf.CellFormat(contentW, 9, titre, "", 1, "L", false, 0, "")
	pdf.Ln(1)

	tableW := recapNomW + float64(len(bloc))*recapColW + recapTauxW

	// Bande matière : une cellule par suite de séances d'une même matière.
	pdf.SetX(pdfMarginL)
	pdf.SetFont("Helvetica", "B", 6)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(recapNomW, 5, "", "", 0, "L", false, 0, "")
	for _, b := range blocsMatiere(bloc) {
		pdf.SetFillColor(80, 72, 160)
		pdf.CellFormat(float64(b.Nb)*recapColW, 5, tr(tronque(b.Nom, b.Nb*3)), "", 0, "C", true, 0, "")
	}
	pdf.CellFormat(recapTauxW, 5, "", "", 1, "C", false, 0, "")

	// En-tête : dates jour/mois.
	pdf.SetX(pdfMarginL)
	pdf.SetFillColor(50, 45, 115)
	pdf.SetFont("Helvetica", "B", 7)
	pdf.CellFormat(recapNomW, 6, "  "+tr("ÉLÈVE"), "", 0, "L", true, 0, "")
	pdf.SetFont("Helvetica", "", 5.5)
	for _, s := range bloc {
		jour := ""
		if !s.StartsAt.IsZero() {
			jour = s.StartsAt.In(parisLoc).Format("02/01")
		}
		pdf.CellFormat(recapColW, 6, jour, "", 0, "C", true, 0, "")
	}
	pdf.SetFont("Helvetica", "B", 6)
	pdf.CellFormat(recapTauxW, 6, "TAUX", "", 1, "C", true, 0, "")

	// Lignes élèves.
	pdf.SetLineWidth(0.1)
	pdf.SetDrawColor(220, 220, 232)
	for i, e := range eleves {
		if pdf.GetY() > pageBasUtile(pdf) {
			pdf.AddPageFormat(orientationRecap(contentW), formatA4)
			pdf.SetX(pdfMarginL)
			pdf.SetFillColor(50, 45, 115)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Helvetica", "B", 7)
			pdf.CellFormat(tableW, 6, "  "+tr("ÉLÈVE (suite)"), "", 1, "L", true, 0, "")
		}

		if i%2 == 0 {
			pdf.SetFillColor(251, 250, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetX(pdfMarginL)
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(20, 20, 20)
		pdf.CellFormat(recapNomW, recapRowH, "  "+tr(tronque(e.Surname+" "+e.Name, 28)), "B", 0, "L", true, 0, "")

		var presents, retards, inscrits int
		pdf.SetFont("Helvetica", "B", 6.5)
		for _, s := range bloc {
			lettre, r, g, b := celluleStatut(statuts[e.UserID][s.SeanceID])
			switch lettre {
			case "P":
				presents++
				inscrits++
			case "R":
				retards++
				inscrits++
			case "A":
				inscrits++
			}
			pdf.SetTextColor(r, g, b)
			pdf.CellFormat(recapColW, recapRowH, lettre, "B", 0, "C", true, 0, "")
		}

		pdf.SetFont("Helvetica", "B", 7)
		pdf.SetTextColor(50, 45, 115)
		taux := "—"
		if inscrits > 0 {
			taux = fmt.Sprintf("%d%%", computeTaux(presents, retards, inscrits))
		}
		pdf.CellFormat(recapTauxW, recapRowH, tr(taux), "B", 1, "C", true, 0, "")
	}

	// Note de convention : le taux ci-dessus sera cité hors contexte.
	pdf.Ln(3)
	pdf.SetX(pdfMarginL)
	pdf.SetFont("Helvetica", "I", 6.5)
	pdf.SetTextColor(120, 120, 135)
	pdf.MultiCell(contentW, 3.2,
		tr("P = présent · R = retard · A = absent · «  – » = séance hors du groupe de l'élève. "+tauxNote),
		"", "L", false)
}

// celluleStatut : lettre et couleur d'une case du récapitulatif.
func celluleStatut(statut string) (string, int, int, int) {
	switch statut {
	case "PRESENT":
		return "P", 22, 130, 60
	case "RETARD":
		return "R", 190, 110, 10
	case "ABSENT":
		return "A", 195, 40, 40
	default:
		// L'élève n'était pas inscrit au groupe de cette séance : la case reste
		// neutre et sort du dénominateur de son taux individuel.
		return "-", 170, 170, 180
	}
}

func tronque(s string, n int) string {
	if n < 1 {
		n = 1
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "."
}

// pageBasUtile : ordonnée à partir de laquelle il faut changer de page, marge
// de pied comprise. Dépend de la hauteur réelle de la page (portrait/paysage).
func pageBasUtile(pdf *fpdf.Fpdf) float64 {
	_, h := pdf.GetPageSize()
	return h - 22
}

// orientationRecap : les pages de continuation d'un bloc doivent conserver
// l'orientation choisie pour le récapitulatif.
func orientationRecap(contentW float64) string {
	if contentW > pdfContentW {
		return "L"
	}
	return "P"
}

// ─── Annexe registre d'intégrité ──────────────────────────────────────────────

func renderLedgerAnnexe(pdf *fpdf.Fpdf, tr func(string) string, doc pdfDoc) {
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(50, 45, 115)
	pdf.CellFormat(pdfContentW, 9, tr("Annexe — registre d'intégrité"), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(70, 70, 85)
	pdf.MultiCell(pdfContentW, 4,
		tr("Chaque pointage est inscrit dans un registre append-only dont chaque maillon "+
			"dépend cryptographiquement du précédent. L'empreinte ci-dessous est celle du "+
			"dernier maillon de la séance ; l'ancre est le premier horodatage RFC 3161 de "+
			"rang supérieur ou égal, qui scelle par chaînage tous les maillons antérieurs. "+
			"La vérification s'effectue depuis l'écran Vérification du témoin."),
		"", "L", false)
	pdf.Ln(3)

	parSeance := make(map[int64]pdfLedgerRow, len(doc.Ledger))
	for _, l := range doc.Ledger {
		parSeance[l.SeanceID] = l
	}

	colW := [5]float64{44, 26, 20, 48, 42}
	entetes := [5]string{tr("SÉANCE"), "MAILLONS", "RANGS", tr("EMPREINTE FINALE"), "ANCRAGE TSA"}

	entete := func() {
		pdf.SetX(pdfMarginL)
		pdf.SetFillColor(50, 45, 115)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 7)
		for i, h := range entetes {
			align := "C"
			if i == 0 || i == 3 {
				align = "L"
			}
			pdf.CellFormat(colW[i], 6, "  "+h, "", 0, align, true, 0, "")
		}
		pdf.Ln(-1)
	}
	entete()

	pdf.SetLineWidth(0.1)
	pdf.SetDrawColor(220, 220, 232)
	for i, s := range doc.Seances {
		if pdf.GetY() > pageBasUtile(pdf) {
			pdf.AddPage()
			entete()
		}
		if i%2 == 0 {
			pdf.SetFillColor(251, 250, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		// Valeurs composées en UTF-8 brut : tr est appliqué une seule fois, au
		// rendu de chaque cellule. Les composer déjà traduites obligerait à ne
		// jamais oublier un tirage cadratin ou une ellipse au passage.
		l, ok := parSeance[s.SeanceID]
		maillons, rangs, hash, ancre := "0", "—", "aucun pointage enregistré", "—"
		if ok {
			maillons = fmt.Sprintf("%d", l.NbMaillons)
			rangs = fmt.Sprintf("%d–%d", l.SeqMin, l.SeqMax)
			hash = tronqueHash(l.Hash)
			if l.AncreSeq > 0 {
				ancre = fmt.Sprintf("#%d · %s", l.AncreSeq, l.AncreAt)
			} else {
				ancre = "non ancré"
			}
		}

		jour := ""
		if !s.StartsAt.IsZero() {
			jour = s.StartsAt.In(parisLoc).Format("02/01")
		}

		pdf.SetX(pdfMarginL)
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(20, 20, 20)
		pdf.CellFormat(colW[0], 5, "  "+tr(tronque(fmt.Sprintf("#%d %s %s", s.SeanceID, jour, s.Matiere), 30)), "B", 0, "L", true, 0, "")
		pdf.SetTextColor(70, 70, 85)
		pdf.CellFormat(colW[1], 5, maillons, "B", 0, "C", true, 0, "")
		pdf.CellFormat(colW[2], 5, tr(rangs), "B", 0, "C", true, 0, "")
		pdf.SetFont("Courier", "", 6.5)
		pdf.CellFormat(colW[3], 5, "  "+tr(hash), "B", 0, "L", true, 0, "")
		pdf.SetFont("Helvetica", "", 6.5)
		if ok && l.AncreSeq == 0 {
			pdf.SetTextColor(180, 95, 10)
		}
		pdf.CellFormat(colW[4], 5, tr(ancre), "B", 1, "C", true, 0, "")
	}
}

// tronqueHash : les 16 premiers hexadigits suffisent à un rapprochement visuel,
// l'empreinte complète restant dans le registre.
func tronqueHash(h string) string {
	h = strings.TrimSpace(h)
	if len(h) <= 16 {
		return h
	}
	return h[:16] + "…"
}
