package bullettin

// Génération des bulletins : lecture du .xlsx et écriture des .docx.
// N'utilise que la bibliothèque standard (aucun Word/Excel requis).
//
// Cette version ne gère QUE les champs Word (Ctrl+F9). Le template doit donc
// utiliser des champs — plus de placeholders entre crochets.
//   Champ « C »            -> colonne C, ligne de l'élève courant
//   Champ « C3 » / « AF »  -> cellule fixe (colonne + ligne, ou colonne seule)
//   Champ « Credit(C3,0) » -> valeur de C3, ou 0 si la note du même module (C) vaut F

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Lecture du classeur .xlsx
// ---------------------------------------------------------------------------

type classeur struct {
	order  []string
	sheets map[string]*feuille
}

type feuille struct {
	valeurs map[string]string
}

func (f *feuille) cell(col string, row int) string {
	return f.valeurs[col+strconv.Itoa(row)]
}

func ouvrirClasseur(path string) (*classeur, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}

	shared, err := lireSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return nil, err
	}
	names, targets, err := lireWorkbook(files)
	if err != nil {
		return nil, err
	}

	book := &classeur{sheets: map[string]*feuille{}}
	for name, target := range targets {
		feuil, err := lireFeuille(files["xl/"+target], shared)
		if err != nil {
			return nil, fmt.Errorf("feuille %q : %w", name, err)
		}
		book.sheets[name] = feuil
	}
	book.order = names
	return book, nil
}

func lireSharedStrings(f *zip.File) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	data, err := lireZip(f)
	if err != nil {
		return nil, err
	}
	type tNode struct {
		Text string `xml:",chardata"`
	}
	type si struct {
		T tNode   `xml:"t"`
		R []tNode `xml:"r>t"`
	}
	var sst struct {
		Si []si `xml:"si"`
	}
	if err := xml.Unmarshal(data, &sst); err != nil {
		return nil, err
	}
	out := make([]string, len(sst.Si))
	for i, s := range sst.Si {
		if len(s.R) > 0 {
			var b strings.Builder
			for _, part := range s.R {
				b.WriteString(part.Text)
			}
			out[i] = b.String()
		} else {
			out[i] = s.T.Text
		}
	}
	return out, nil
}

func lireWorkbook(files map[string]*zip.File) ([]string, map[string]string, error) {
	wbData, err := lireZip(files["xl/workbook.xml"])
	if err != nil {
		return nil, nil, err
	}
	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := xml.Unmarshal(wbData, &wb); err != nil {
		return nil, nil, err
	}

	relsData, err := lireZip(files["xl/_rels/workbook.xml.rels"])
	if err != nil {
		return nil, nil, err
	}
	var rels struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.Unmarshal(relsData, &rels); err != nil {
		return nil, nil, err
	}
	idToTarget := map[string]string{}
	for _, rel := range rels.Rel {
		idToTarget[rel.ID] = strings.TrimPrefix(rel.Target, "/xl/")
	}

	var order []string
	targets := map[string]string{}
	for _, s := range wb.Sheets {
		order = append(order, s.Name)
		if t, ok := idToTarget[s.RID]; ok {
			targets[s.Name] = t
		}
	}
	return order, targets, nil
}

func lireFeuille(f *zip.File, shared []string) (*feuille, error) {
	if f == nil {
		return &feuille{valeurs: map[string]string{}}, nil
	}
	data, err := lireZip(f)
	if err != nil {
		return nil, err
	}
	var ws struct {
		Rows []struct {
			Cells []struct {
				Ref    string `xml:"r,attr"`
				Type   string `xml:"t,attr"`
				V      string `xml:"v"`
				Inline struct {
					Text string `xml:"t"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(data, &ws); err != nil {
		return nil, err
	}

	vals := map[string]string{}
	for _, row := range ws.Rows {
		for _, c := range row.Cells {
			var v string
			switch c.Type {
			case "s":
				if idx, err := strconv.Atoi(strings.TrimSpace(c.V)); err == nil && idx >= 0 && idx < len(shared) {
					v = shared[idx]
				}
			case "inlineStr":
				v = c.Inline.Text
			default:
				v = c.V
			}
			vals[c.Ref] = v
		}
	}
	return &feuille{valeurs: vals}, nil
}

// ---------------------------------------------------------------------------
// Champs Word (Ctrl+F9)
// ---------------------------------------------------------------------------
//
// Un champ complexe est une suite de runs : run(fldChar begin) … run(instrText)
// … run(fldChar end). Le code tapé est dans les <w:instrText> (parfois fragmenté,
// mais toujours à l'intérieur du champ : il suffit de les concaténer).
// On remplace le champ entier par un run texte contenant la valeur calculée.
//
// Balayage manuel (pas de grosse regex) car en RE2 un <w:rPr>.*?</w:rPr> peut
// traverser les frontières de runs et fausser la capture.

var (
	balBegin  = []byte(`<w:fldChar w:fldCharType="begin"/>`)
	balEnd    = []byte(`<w:fldChar w:fldCharType="end"/>`)
	balRunFin = []byte(`</w:r>`)
)

// Un run peut porter des attributs : <w:r> ou <w:r w:rsidR="...">.
var reRunOpen = regexp.MustCompile(`<w:r(?: [^>]*)?>`)
var reInstr = regexp.MustCompile(`<w:instrText[^>]*>([^<]*)</w:instrText>`)
var reRpr = regexp.MustCompile(`(?s)<w:rPr>.*?</w:rPr>`)

// Codes de champ reconnus.
var reCreditCode = regexp.MustCompile(`^Credit\(([A-Za-z]+[0-9]+),([^)]*)\)$`)
var reCellCode = regexp.MustCompile(`^([A-Za-z]+)([0-9]*)$`)
var reLettres = regexp.MustCompile(`^[A-Za-z]+`)

func remplacerChamps(content []byte, f *feuille, ligneEleve int) []byte {
	var out bytes.Buffer
	i := 0
	for {
		b := bytes.Index(content[i:], balBegin)
		if b < 0 {
			out.Write(content[i:])
			break
		}
		b += i

		// Début du run contenant le fldChar begin (dernier run ouvert dans [i:b]).
		opens := reRunOpen.FindAllIndex(content[i:b], -1)
		if len(opens) == 0 {
			out.Write(content[i:])
			break
		}
		runStart := i + opens[len(opens)-1][0]

		// fldChar end correspondant (pas d'imbrication dans ces bulletins).
		e := bytes.Index(content[b:], balEnd)
		if e < 0 {
			out.Write(content[i:])
			break
		}
		e += b
		fin := bytes.Index(content[e:], balRunFin)
		if fin < 0 {
			out.Write(content[i:])
			break
		}
		endRunEnd := e + fin + len(balRunFin)

		// Code du champ = concaténation des instrText entre begin et end.
		var code []byte
		for _, im := range reInstr.FindAllSubmatch(content[b:e], -1) {
			code = append(code, im[1]...)
		}

		// Formatage : le rPr du run de début (recherché dans ce seul run).
		rpr := reRpr.Find(content[runStart:b])

		out.Write(content[i:runStart]) // tout ce qui précède le champ

		if val, ok := valeurCode(string(code), f, ligneEleve); ok {
			out.WriteString(`<w:r>`)
			out.Write(rpr)
			out.WriteString(`<w:t xml:space="preserve">`)
			out.WriteString(escapeXML(val))
			out.WriteString(`</w:t></w:r>`)
		} else {
			out.Write(content[runStart:endRunEnd]) // code non géré -> champ laissé intact
		}

		i = endRunEnd
	}
	return out.Bytes()
}

// valeurCode calcule la valeur d'un code de champ. Renvoie ok=false si le code
// n'est pas reconnu (le champ est alors laissé tel quel).
func valeurCode(code string, f *feuille, ligneEleve int) (string, bool) {
	code = strings.TrimSpace(code)

	// Credit(ref,repli) : 0 (ou repli) si la note du même module vaut F.
	if mm := reCreditCode.FindStringSubmatch(code); mm != nil {
		creditRef, repli := mm[1], mm[2]
		gradeCol, _ := separerRef(creditRef)
		note := strings.TrimSpace(f.cell(gradeCol, ligneEleve))
		if strings.EqualFold(note, "F") {
			return repli, true
		}
		col, row := separerRef(creditRef)
		return cellText(f, col, row), true
	}

	// Référence de cellule : lettres + chiffres optionnels. Sans chiffre -> ligne élève.
	if mm := reCellCode.FindStringSubmatch(code); mm != nil {
		col, digits := mm[1], mm[2]
		row := ligneEleve
		if digits != "" {
			row, _ = strconv.Atoi(digits)
		}
		return cellText(f, col, row), true
	}

	return "", false
}

// separerRef découpe "C3" en colonne "C" et ligne 3.
func separerRef(ref string) (string, int) {
	col := reLettres.FindString(ref)
	row, _ := strconv.Atoi(ref[len(col):])
	return col, row
}

// ---------------------------------------------------------------------------
// Génération d'un bulletin .docx
// ---------------------------------------------------------------------------

func genererBulletin(templatePath, destPath string, f *feuille, ligneEleve int) error {
	zr, err := zip.OpenReader(templatePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, file := range zr.File {
		contenu, err := lireZip(file)
		if err != nil {
			return err
		}
		if strings.HasPrefix(file.Name, "word/") && strings.HasSuffix(file.Name, ".xml") {
			contenu = remplacerChamps(contenu, f, ligneEleve)
		}
		w, err := zw.Create(file.Name)
		if err != nil {
			return err
		}
		if _, err := w.Write(contenu); err != nil {
			return err
		}
	}
	return nil
}

func cellText(f *feuille, col string, row int) string {
	v := f.cell(col, row)
	if v == "" {
		return ""
	}
	if x, err := strconv.ParseFloat(strings.Replace(v, ",", ".", 1), 64); err == nil {
		if col == "N" || col == "O" {
			return strings.Replace(strconv.FormatFloat(x, 'f', 2, 64), ".", ",", 1)
		}
		return strconv.FormatInt(int64(math.Round(x)), 10)
	}
	return v
}

// ---------------------------------------------------------------------------
// Utilitaires
// ---------------------------------------------------------------------------

func lireZip(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

var reBadChars = regexp.MustCompile(`[\\/:*?"<>|]`)

func sanitize(s string) string {
	return reBadChars.ReplaceAllString(s, "")
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
