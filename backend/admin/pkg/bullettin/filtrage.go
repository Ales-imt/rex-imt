package bullettin

import (
	"sort"
	"strings"
)

// ColonneInfo : une colonne et son en-tête (ligne 2), pour la liste de filtrage.
type ColonneInfo struct {
	Letter string `json:"letter"`
	Header string `json:"header"`
}

// ---------------------------------------------------------------------------
// Aide au filtrage (fonctions libres, testables indépendamment du serveur HTTP)
// ---------------------------------------------------------------------------

// filtrePasse : vrai si la ligne doit être générée. Filtre vide -> tout passe.
func filtrePasse(f *feuille, filterCol string, r int) bool {
	if filterCol == "" {
		return true
	}
	return strings.TrimSpace(f.cell(filterCol, r)) != "-"
}

// listerColonnes renvoie les colonnes ayant un en-tête en ligne 2, triées.
func listerColonnes(f *feuille) []ColonneInfo {
	const headerRow = 2
	type item struct {
		idx  int
		info ColonneInfo
	}
	var items []item
	seen := map[string]bool{}
	for ref := range f.valeurs {
		col, row := separerRef(ref)
		if row != headerRow || col == "" || seen[col] {
			continue
		}
		seen[col] = true
		h := strings.ReplaceAll(strings.TrimSpace(f.cell(col, headerRow)), "\n", " ")
		items = append(items, item{colIndex(col), ColonneInfo{Letter: col, Header: h}})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].idx < items[j].idx })
	out := make([]ColonneInfo, len(items))
	for i, it := range items {
		out[i] = it.info
	}
	return out
}

// colIndex convertit une lettre de colonne en indice ("A"->1, "AA"->27).
func colIndex(col string) int {
	n := 0
	for _, ch := range strings.ToUpper(col) {
		n = n*26 + int(ch-'A'+1)
	}
	return n
}

func errBool(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
