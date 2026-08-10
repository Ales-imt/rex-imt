package presence

// Tests internes (package presence) : generateSemestrePDF n'est pas exportée.
// Les tests de bout en bout du registre restent dans presence_test.

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"
)

// ─── Outils d'inspection du PDF produit ───────────────────────────────────────

// pdfPages découpe le document en flux de contenu, un par page, dans l'ordre du
// fichier (fpdf émet les pages séquentiellement). Les flux sont compressés en
// zlib ; seuls les contenus de page le sont ici, les polices étant les polices
// de base non embarquées.
func pdfPages(t *testing.T, raw []byte) []string {
	t.Helper()

	var pages []string
	rest := raw
	for {
		i := bytes.Index(rest, []byte("stream"))
		if i < 0 {
			break
		}
		body := rest[i+len("stream"):]
		body = bytes.TrimLeft(body, "\r\n")
		j := bytes.Index(body, []byte("endstream"))
		if j < 0 {
			break
		}
		zr, err := zlib.NewReader(bytes.NewReader(body[:j]))
		if err == nil {
			if out, err := io.ReadAll(zr); err == nil {
				pages = append(pages, string(out))
			}
			zr.Close()
		}
		// Reprendre APRÈS le marqueur : repartir dessus ferait re-matcher le
		// suffixe « stream » de « endstream » et sauter la page suivante.
		rest = body[j+len("endstream"):]
	}
	return pages
}

// nbPages compte les objets page, indépendamment des flux de contenu.
func nbPages(raw []byte) int {
	return bytes.Count(raw, []byte("/Type /Page\n"))
}

// matiereIDs attribue un identifiant stable à chaque nom de matière : le
// regroupement du sommaire et du récapitulatif se fait sur MatiereID, pas sur
// le libellé.
var matiereIDs = map[string]int64{}

func matiereID(nom string) int64 {
	if id, ok := matiereIDs[nom]; ok {
		return id
	}
	id := int64(len(matiereIDs) + 1)
	matiereIDs[nom] = id
	return id
}

// seanceFixture fabrique une séance de n élèves, dont un tiers en retard et un
// autre tiers absent, afin que les trois statuts soient rendus. Les élèves
// portent des UserID stables d'une séance à l'autre : c'est cette clé qui
// permet au récapitulatif de reconstituer une ligne par élève.
func seanceFixture(id int64, code, matiere string, n int) pdfData {
	eleves := make([]pdfEleveRow, 0, n)
	var presents, retards, absents int
	for i := 0; i < n; i++ {
		statut := "PRESENT"
		switch i % 3 {
		case 1:
			statut = "RETARD"
		case 2:
			statut = "ABSENT"
		}
		switch statut {
		case "PRESENT":
			presents++
		case "RETARD":
			retards++
		default:
			absents++
		}
		eleves = append(eleves, pdfEleveRow{
			Rank:     i + 1,
			UserID:   int32(1000 + i),
			Surname:  fmt.Sprintf("Nom%02d", i),
			Name:     fmt.Sprintf("Prenom%02d", i),
			Statut:   statut,
			PointeAt: "08:15",
		})
	}
	return pdfData{
		SeanceID: id, Code: code, Matiere: matiere,
		MatiereID: matiereID(matiere),
		StartsAt:  time.Date(2026, 1, 12, 8, 0, 0, 0, time.UTC).AddDate(0, 0, int(id)),
		Promo:     "ING1", Prof: "Dupont", Salle: "A101",
		Date: "Lundi 12 janvier 2026", Horaire: "08:00 – 10:00",
		OpenedAt: "07:58", ClosedAt: "10:02", LateMin: 10,
		Total: n, Presents: presents, Retards: retards, Absents: absents,
		Taux:        computeTaux(presents, retards, n, 0),
		Eleves:      eleves,
		GeneratedAt: time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC),
	}
}

func docFixture(seances ...pdfData) pdfDoc {
	return pdfDoc{
		Promo: "ING1", Periode: "S2", Annee: 2025,
		GeneratedAt: time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC),
		Seances:     seances,
		Options:     pdfOptions{},
	}
}

func generate(t *testing.T, doc pdfDoc) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := generateSemestrePDF(&buf, doc); err != nil {
		t.Fatalf("generateSemestrePDF: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("sortie non reconnue comme un PDF")
	}
	return buf.Bytes()
}

// ─── Cas ──────────────────────────────────────────────────────────────────────

// Un semestre sans séance éditable doit tout de même produire un PDF valide :
// un document à zéro page ne s'ouvre pas.
func TestGenerateSemestrePDF_ZeroSeance(t *testing.T) {
	raw := generate(t, docFixture())

	if got := nbPages(raw); got != 1 {
		t.Fatalf("pages = %d, attendu 1", got)
	}
	if !strings.Contains(strings.Join(pdfPages(t, raw), ""), "Aucune s") {
		t.Error("la page unique doit annoncer l'absence de séance")
	}
}

func TestGenerateSemestrePDF_UneSeance(t *testing.T) {
	raw := generate(t, docFixture(seanceFixture(12, "AAA111", "Maths", 20)))

	if got := nbPages(raw); got != 1 {
		t.Fatalf("pages = %d, attendu 1", got)
	}
	page := pdfPages(t, raw)[0]
	for _, attendu := range []string{"#12", "AAA111", "Nom00", "Prenom19"} {
		if !strings.Contains(page, attendu) {
			t.Errorf("la feuille ne contient pas %q", attendu)
		}
	}
}

// La colonne signature est optionnelle ; son en-tête ne doit apparaître que
// lorsqu'elle est demandée.
func TestGenerateSemestrePDF_ColonneSignature(t *testing.T) {
	for _, tc := range []struct {
		nom      string
		opt      bool
		attendue bool
	}{
		{"sans", false, false},
		{"avec", true, true},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			doc := docFixture(seanceFixture(1, "AAA111", "Maths", 5))
			doc.Options.Signature = tc.opt
			page := pdfPages(t, generate(t, doc))[0]

			if got := strings.Contains(page, "SIGNATURE"); got != tc.attendue {
				t.Errorf("colonne SIGNATURE présente = %v, attendu %v", got, tc.attendue)
			}
		})
	}
}

func TestGenerateSemestrePDF_PlusieursMatieres(t *testing.T) {
	raw := generate(t, docFixture(
		seanceFixture(1, "AAA111", "Maths", 12),
		seanceFixture(2, "BBB222", "Maths", 12),
		seanceFixture(3, "CCC333", "Physique", 12),
	))

	if got := nbPages(raw); got != 3 {
		t.Fatalf("pages = %d, attendu 3 (une par séance)", got)
	}
	pages := pdfPages(t, raw)
	for i, code := range []string{"AAA111", "BBB222", "CCC333"} {
		if !strings.Contains(pages[i], code) {
			t.Errorf("page %d : code %q absent", i+1, code)
		}
	}
}

// Cœur du refactor : le pied de page lit un pointeur mis à jour avant chaque
// AddPage. Capturé par valeur, il ferait porter la première séance à toutes les
// pages du document — c'est précisément ce que ce test interdit.
func TestGenerateSemestrePDF_PiedDePageParSeance(t *testing.T) {
	raw := generate(t, docFixture(
		seanceFixture(101, "AAA111", "Maths", 8),
		seanceFixture(202, "BBB222", "Physique", 8),
		seanceFixture(303, "CCC333", "Chimie", 8),
	))

	pages := pdfPages(t, raw)
	if len(pages) != 3 {
		t.Fatalf("flux de page = %d, attendu 3", len(pages))
	}

	codes := []string{"AAA111", "BBB222", "CCC333"}
	for i, page := range pages {
		if !strings.Contains(page, codes[i]) {
			t.Errorf("page %d : pied attendu avec %q", i+1, codes[i])
		}
		for j, autre := range codes {
			if j != i && strings.Contains(page, autre) {
				t.Errorf("page %d : porte le code %q d'une autre séance", i+1, autre)
			}
		}
	}
}

// La pagination doit être continue sur tout le document, y compris à travers
// les pages de débordement du tableau élèves — et non repartir à 1 par séance.
func TestGenerateSemestrePDF_PaginationContinue(t *testing.T) {
	// 40 élèves débordent sur une deuxième page (l'en-tête et la barre de
	// statistiques laissent place à ~23 lignes sur la première) ; deux séances
	// donnent donc quatre pages, numérotées de 1 à 4 et non 1-2 puis 1-2.
	raw := generate(t, docFixture(
		seanceFixture(1, "AAA111", "Maths", 40),
		seanceFixture(2, "BBB222", "Physique", 40),
	))

	total := nbPages(raw)
	if total != 4 {
		t.Fatalf("pages = %d, attendu 4 (2 séances × 2 pages)", total)
	}

	pages := pdfPages(t, raw)
	for i, page := range pages {
		attendu := fmt.Sprintf("Page %d/%d", i+1, total)
		if !strings.Contains(page, attendu) {
			t.Errorf("page %d : pied attendu %q", i+1, attendu)
		}
	}

	// La page de débordement d'une séance porte le code de CETTE séance.
	if !strings.Contains(pages[1], "AAA111") {
		t.Error("page 2 (débordement de la séance 1) : code AAA111 attendu")
	}
	if !strings.Contains(pages[2], "BBB222") {
		t.Error("page 3 (début de la séance 2) : code BBB222 attendu")
	}
}

// ─── Sections ─────────────────────────────────────────────────────────────────

// Une séance non clôturée ne doit pas faire échouer le document : elle sort du
// corps mais doit être nommée en page de garde, faute de quoi son absence
// passerait inaperçue.
func TestGenerateSemestrePDF_SeancesExcluesEnPageDeGarde(t *testing.T) {
	doc := docFixture(
		seanceFixture(1, "AAA111", "Maths", 10),
		seanceFixture(2, "BBB222", "Maths", 10),
	)
	doc.Options.Cover = true
	doc.Exclues = []string{
		"Maths - lundi 19 janvier 2026 - 08:00 a 10:00",
		"Physique - mardi 20 janvier 2026 - 14:00 a 16:00",
	}
	raw := generate(t, doc)

	// 1 garde + 1 sommaire + 2 feuilles ; les exclues n'ajoutent pas de feuille.
	if got := nbPages(raw); got != 4 {
		t.Fatalf("pages = %d, attendu 4 (garde + sommaire + 2 feuilles)", got)
	}

	garde := pdfPages(t, raw)[0]
	if !strings.Contains(garde, "2 s") {
		t.Error("la page de garde doit annoncer le nombre de séances exclues")
	}
	for _, e := range doc.Exclues {
		if !strings.Contains(garde, e) {
			t.Errorf("séance exclue non listée en page de garde : %q", e)
		}
	}

	// Aucune feuille ne doit avoir été produite pour les séances exclues.
	for _, page := range pdfPages(t, raw) {
		if strings.Contains(page, "Physique") && !strings.Contains(page, "non cl") {
			t.Error("une séance exclue a produit une feuille")
		}
	}
}

// Un semestre dont rien n'est clôturé ne doit pas produire des dizaines de
// pages de puces : la liste est plafonnée, mais le total reste annoncé.
func TestGenerateSemestrePDF_ExclusionsPlafonnees(t *testing.T) {
	doc := docFixture(seanceFixture(1, "AAA111", "Maths", 10))
	doc.Options.Cover = true
	for i := 0; i < maxExcluesNommees+15; i++ {
		doc.Exclues = append(doc.Exclues, fmt.Sprintf("Matiere %03d - 12/01/2026", i))
	}
	raw := generate(t, doc)

	garde := strings.Join(pdfPages(t, raw), "")
	if !strings.Contains(garde, "Matiere 000") {
		t.Error("les premières séances exclues doivent être nommées")
	}
	if strings.Contains(garde, fmt.Sprintf("Matiere %03d", maxExcluesNommees)) {
		t.Errorf("la liste doit s'arrêter à %d entrées nommées", maxExcluesNommees)
	}
	if !strings.Contains(garde, "et 15 autre") {
		t.Error("le reste non détaillé doit être dénombré")
	}
	if !strings.Contains(garde, fmt.Sprintf("%d s", len(doc.Exclues))) {
		t.Error("le total des séances exclues doit rester annoncé")
	}
}

// Le sommaire est composé avant les feuilles : ses numéros de page sont écrits
// sous forme de jetons, qui doivent tous être résolus dans le document final.
func TestGenerateSemestrePDF_SommaireNumerosDePage(t *testing.T) {
	doc := docFixture(
		seanceFixture(1, "AAA111", "Maths", 10),
		seanceFixture(2, "BBB222", "Maths", 10),
		seanceFixture(3, "CCC333", "Physique", 10),
	)
	doc.Options.Cover = true
	raw := generate(t, doc)

	if bytes.Contains(raw, []byte("{{p")) {
		t.Error("un jeton de numéro de page du sommaire n'a pas été résolu")
	}

	// Garde (1), sommaire (2), puis Maths en 3-4 et Physique en 5.
	sommaire := pdfPages(t, raw)[1]
	if !strings.Contains(sommaire, "Maths") || !strings.Contains(sommaire, "Physique") {
		t.Fatal("le sommaire doit lister les deux matières")
	}
	for _, attendu := range []string{"3", "5"} {
		if !strings.Contains(sommaire, attendu) {
			t.Errorf("le sommaire ne porte pas le numéro de page %q", attendu)
		}
	}
}

// Le récapitulatif croise les élèves et les séances ; au-delà du seuil il passe
// en paysage, ce qui ne doit pas contaminer l'orientation des feuilles.
func TestGenerateSemestrePDF_Recap(t *testing.T) {
	var seances []pdfData
	for i := 0; i < 15; i++ {
		seances = append(seances, seanceFixture(int64(i+1), fmt.Sprintf("C%05d", i), "Maths", 6))
	}
	doc := docFixture(seances...)
	doc.Options.Recap = true
	raw := generate(t, doc)

	pages := pdfPages(t, raw)
	recap := pages[0]
	if !strings.Contains(recap, "capitulatif") {
		t.Fatal("la première page doit être le récapitulatif")
	}
	if !strings.Contains(recap, "Nom00") {
		t.Error("le récapitulatif doit lister les élèves")
	}
	if !strings.Contains(recap, "TAUX") {
		t.Error("le récapitulatif doit porter la colonne de taux")
	}
	if !strings.Contains(recap, "pr") { // extrait de la note de convention
		t.Error("le récapitulatif doit porter la note de convention du taux")
	}

	// Le récapitulatif est en paysage (15 > recapSeuilPaysage), les feuilles non.
	if !bytes.Contains(raw, []byte("/MediaBox [0 0 841.89 595.28]")) {
		t.Error("le récapitulatif devrait être composé en paysage")
	}
	if !bytes.Contains(raw, []byte("/MediaBox [0 0 595.28 841.89]")) {
		t.Error("les feuilles de séance doivent rester en portrait")
	}
}

func TestGenerateSemestrePDF_AnnexeLedger(t *testing.T) {
	doc := docFixture(
		seanceFixture(1, "AAA111", "Maths", 10),
		seanceFixture(2, "BBB222", "Maths", 10),
	)
	doc.Options.Ledger = true
	doc.Ledger = []pdfLedgerRow{
		{SeanceID: 1, SeqMin: 10, SeqMax: 19, NbMaillons: 10,
			Hash: "abcdef0123456789abcdef0123456789", AncreSeq: 25, AncreAt: "01/02/2026"},
		// Séance 2 sans ancre : le document doit le dire, pas le taire.
		{SeanceID: 2, SeqMin: 20, SeqMax: 29, NbMaillons: 10,
			Hash: "99887766554433221100aabbccddeeff"},
	}
	raw := generate(t, doc)

	annexe := pdfPages(t, raw)[2]
	if !strings.Contains(annexe, "Annexe") {
		t.Fatal("la dernière page doit être l'annexe registre")
	}
	for _, attendu := range []string{"abcdef0123456789", "10", "19", "25", "non anc"} {
		if !strings.Contains(annexe, attendu) {
			t.Errorf("l'annexe ne contient pas %q", attendu)
		}
	}
}

// ─── Encodage ─────────────────────────────────────────────────────────────────

// Les polices de base ne sont pas unicode : tout texte doit traverser le
// traducteur cp1252 exactement une fois. L'oublier laisse passer des octets
// utf-8 bruts (« â€” » à la place d'un cadratin) ; l'appliquer deux fois
// remplace chaque accent par un point (« Pr.sents »). Les deux défauts ont
// existé, aucun ne fait échouer la génération : seul le document les révèle.
func TestGenerateSemestrePDF_EncodageCP1252(t *testing.T) {
	doc := docFixture(
		seanceFixture(1, "AAA111", "Mathématiques appliquées", 6),
		seanceFixture(2, "BBB222", "Développement", 6),
		seanceFixture(3, "CCC333", "Développement", 6),
	)
	doc.Options = pdfOptions{Cover: true, Recap: true, Signature: true, Ledger: true}
	doc.Exclues = []string{"Séance #42 du 12/03 — Développement"}
	doc.Ledger = []pdfLedgerRow{
		// Une séance ancrée, une sans ancre, une absente du registre : les trois
		// branches de l'annexe composent chacune leurs propres libellés.
		{SeanceID: 1, SeqMin: 6, SeqMax: 9, NbMaillons: 4,
			Hash: "bfbcbff7d6cf00cd1122334455667788", AncreSeq: 6, AncreAt: "07/08/2026"},
		{SeanceID: 2, SeqMin: 10, SeqMax: 13, NbMaillons: 4,
			Hash: "1676f482ca12f5471122334455667788"},
		// La séance 3 est volontairement hors registre.
	}
	raw := generate(t, doc)

	for i, page := range pdfPages(t, raw) {
		if j := indexUTF8Multibyte(page); j >= 0 {
			t.Errorf("page %d : utf-8 non traduit à l'offset %d : %q",
				i+1, j, extrait(page, j))
		}
	}

	// Une double traduction ne laisse pas d'octet suspect, seulement des points :
	// on vérifie que des libellés accentués de chaque section sont intacts.
	tr := fpdf.New("P", "mm", "A4", "").UnicodeTranslatorFromDescriptor("")
	tout := strings.Join(pdfPages(t, raw), "")
	for _, libelle := range []string{
		"Année universitaire",       // page de garde
		"Sommaire des matières",     // sommaire
		"Présents",                  // feuille de séance
		"Mathématiques appliquées",  // donnée métier
		"aucun pointage enregistré", // annexe registre
	} {
		if !strings.Contains(tout, tr(libelle)) {
			t.Errorf("libellé %q absent ou doublement traduit", libelle)
		}
	}
}

// indexUTF8Multibyte repère la première amorce d'une séquence utf-8 multi-octets
// (0xC2–0xF4 suivi d'un octet de continuation). En cp1252 une telle paire est du
// texte accidentel — « Ã© », « â€” » — et non un accent voulu.
func indexUTF8Multibyte(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] >= 0xC2 && s[i] <= 0xF4 && s[i+1] >= 0x80 && s[i+1] <= 0xBF {
			return i
		}
	}
	return -1
}

func extrait(s string, i int) string {
	fin := min(i+12, len(s))
	return s[i:fin]
}

// ─── Nom de fichier ───────────────────────────────────────────────────────────

func TestExportFilename(t *testing.T) {
	for _, tc := range []struct {
		nom     string
		header  GetPeriodeHeaderRow
		attendu string
	}{
		{
			"cas nominal",
			GetPeriodeHeaderRow{ID: 42, PromoName: "ING1", PeriodeName: "S2", Annee: 2025},
			"presence-ing1-s2-2025-2026.pdf",
		},
		{
			"accents et espaces",
			GetPeriodeHeaderRow{ID: 7, PromoName: "Ingénieur 2ème année", PeriodeName: "Semestre 3", Annee: 2026},
			"presence-ingenieur-2eme-annee-semestre-3-2026-2027.pdf",
		},
		{
			// Sans année exploitable, l'identifiant évite deux fichiers homonymes.
			"année absente",
			GetPeriodeHeaderRow{ID: 42, PromoName: "ING1", PeriodeName: "S2"},
			"presence-ing1-s2-42.pdf",
		},
		{
			"libellés vides",
			GetPeriodeHeaderRow{ID: 9, Annee: 2025},
			"presence-2025-2026.pdf",
		},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			if got := exportFilename(tc.header); got != tc.attendu {
				t.Errorf("exportFilename = %q, attendu %q", got, tc.attendu)
			}
		})
	}
}

// ─── Convention de taux ───────────────────────────────────────────────────────

func TestComputeTaux(t *testing.T) {
	for _, tc := range []struct {
		nom                                  string
		presents, retards, inscrits, excuses int
		attendu                              int
	}{
		{"un retard compte comme une présence", 18, 2, 25, 0, 80},
		{"effectif nul", 0, 0, 0, 0, 0},
		{"aucun présent", 0, 0, 30, 0, 0},
		{"tous présents", 30, 0, 30, 0, 100},
		// Le cas qui motivait la convention : des pointages hors groupe ne
		// doivent jamais porter le taux au-delà de 100 %.
		{"plafonné à l'effectif inscrit", 25, 0, 25, 0, 100},
		// Les excusés sortent du DÉNOMINATEUR : sans cela, 24 présents sur 25
		// inscrits dont un excusé donnerait 96 % et pénaliserait la promotion
		// pour une absence pourtant justifiée.
		{"un excusé ne fait pas baisser le taux", 24, 0, 25, 1, 100},
		{"excusés et retard combinés", 20, 2, 25, 3, 100},
		{"promotion intégralement excusée", 0, 0, 25, 25, 0},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			if got := computeTaux(tc.presents, tc.retards, tc.inscrits, tc.excuses); got != tc.attendu {
				t.Errorf("computeTaux(%d, %d, %d, %d) = %d, attendu %d",
					tc.presents, tc.retards, tc.inscrits, tc.excuses, got, tc.attendu)
			}
		})
	}
}

// ─── Excuses ──────────────────────────────────────────────────────────────────

// seanceAvecExcuse : trois élèves, un présent, un absent, un absent excusé.
func seanceAvecExcuse() pdfData {
	rows := []presenceLigne{
		{UserID: 1, Surname: "Alpha", Name: "Anne", Statut: "PRESENT"},
		{UserID: 2, Surname: "Beta", Name: "Bruno", Statut: "ABSENT"},
		{UserID: 3, Surname: "Gamma", Name: "Gil", Statut: "ABSENT", Justifie: true},
		// Un excusé QUI A POINTÉ reste présent : le pointage l'emporte.
		{UserID: 4, Surname: "Delta", Name: "Dan", Statut: "PRESENT", Justifie: true},
	}
	d := buildPdfData(seanceInfo{ID: 1, Code: "AAA111", MatiereName: "Maths"}, rows, nil,
		time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC))
	d.MatiereID = matiereID("Maths")
	d.StartsAt = time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	return d
}

func TestBuildPdfDataCompteLesExcuses(t *testing.T) {
	d := seanceAvecExcuse()

	if d.Presents != 2 {
		t.Errorf("Presents = %d, attendu 2 (l'excuse n'écrase pas un pointage)", d.Presents)
	}
	if d.Absents != 1 {
		t.Errorf("Absents = %d, attendu 1 : l'absent excusé est compté à part", d.Absents)
	}
	if d.Excuses != 1 {
		t.Errorf("Excuses = %d, attendu 1", d.Excuses)
	}
	if d.Presents+d.Retards+d.Absents+d.Excuses != d.Total {
		t.Errorf("les catégories ne partitionnent pas l'effectif : %d+%d+%d+%d != %d",
			d.Presents, d.Retards, d.Absents, d.Excuses, d.Total)
	}
	// 2 présents sur 4 inscrits dont 1 excusé → 2/3.
	if attendu := (2 * 100) / 3; d.Taux != attendu {
		t.Errorf("Taux = %d%%, attendu %d%% (l'excusé sort du dénominateur)", d.Taux, attendu)
	}
}

func TestGenerateSemestrePDF_LibelleExcuse(t *testing.T) {
	doc := docFixture(seanceAvecExcuse())
	raw := generate(t, doc)

	feuille := strings.Join(pdfPages(t, raw), "\n")
	if !strings.Contains(feuille, "Excus") {
		t.Error("la feuille doit porter le libellé « Excusé » et la case de statistiques « Excusés »")
	}
	// Fond jaune pâle de la ligne excusée : 255,247,214 en composantes fpdf.
	if !strings.Contains(feuille, "1.000 0.969 0.839 rg") {
		t.Error("la ligne excusée doit être teintée en jaune pâle, à la place du zébrage")
	}
	// Libellé « Excusé » en 146,90,10.
	if !strings.Contains(feuille, "0.573 0.353 0.039 rg") {
		t.Error("le libellé « Excusé » doit être coloré (146,90,10)")
	}
}

func TestGenerateSemestrePDF_RecapMarqueLesExcuses(t *testing.T) {
	doc := docFixture(seanceAvecExcuse())
	doc.Options.Recap = true
	raw := generate(t, doc)

	recap := pdfPages(t, raw)[0]
	if !strings.Contains(recap, "capitulatif") {
		t.Fatal("la première page doit être le récapitulatif")
	}
	// La matrice n'a qu'une lettre par case : l'excuse y devient « E ».
	if !strings.Contains(recap, "(E)Tj") {
		t.Error("le récapitulatif doit marquer la séance excusée d'un « E »")
	}
}
