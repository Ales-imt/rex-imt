package webdfd

import (
	"back-rex-common/pkg/services"
	"back-rex-sync/pkg/source"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const throttle = 50 * time.Millisecond

// Source interroge cybema en direct : cgiempt.exe pour les référentiels
// et le planning, cgihtml.exe pour la composition des groupes.
type Source struct {
	BaseURL        string // cgiempt.exe (promos, cours, eleves, profs, planning)
	ListeGroupeURL string // cgihtml.exe (listegroupe MODE=10)
}

// NewSource assemble la source webdfd depuis la configuration.
func NewSource(cfg services.WebdfdConfig) *Source {
	return &Source{BaseURL: cfg.BaseURL, ListeGroupeURL: cfg.ListeGroupeURL}
}

// get effectue un GET sur webdfd puis marque une pause pour ne pas saturer le serveur.
func get(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	time.Sleep(throttle)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

// fetchTexte récupère un flux *_txt et le décode depuis Windows-1252.
func fetchTexte(url string) (string, error) {
	resp, err := get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// parseKV découpe une ligne « CLE;valeur;CLE;valeur » des flux *_txt.
func parseKV(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
}

// lignesKV itère les lignes utiles d'un flux *_txt, déjà découpées en clé/valeur.
func lignesKV(body string, f func(kv map[string]string)) {
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		f(parseKV(line))
	}
}

func (s *Source) Promos() ([]source.Promo, error) {
	body, err := fetchTexte(s.BaseURL + "?TYPE=promos_txt")
	if err != nil {
		return nil, fmt.Errorf("webdfd: promos_txt inaccessible: %w", err)
	}

	var promos []source.Promo
	lignesKV(body, func(kv map[string]string) {
		p0 := kv["P0"]
		nom := strings.TrimSpace(kv["NOM"])
		if p0 == "" || nom == "" {
			return
		}
		// Un P0 non numérique est une ligne de service, pas une promotion.
		if _, err := strconv.ParseInt(p0, 10, 64); err != nil {
			return
		}
		promos = append(promos, source.Promo{ExternalID: p0, Nom: nom})
	})
	return promos, nil
}

func (s *Source) Profs() ([]source.Personne, error) {
	body, err := fetchTexte(s.BaseURL + "?TYPE=profs_txt")
	if err != nil {
		return nil, fmt.Errorf("webdfd: profs_txt inaccessible: %w", err)
	}
	return personnes(body, "PR"), nil
}

func (s *Source) Eleves() ([]source.Personne, error) {
	body, err := fetchTexte(s.BaseURL + "?TYPE=eleves_txt")
	if err != nil {
		return nil, fmt.Errorf("webdfd: eleves_txt inaccessible: %w", err)
	}
	return personnes(body, "EV"), nil
}

// personnes extrait les lignes d'un flux profs_txt ou eleves_txt, qui ne
// diffèrent que par le nom de la clé d'identité.
func personnes(body, cle string) []source.Personne {
	var out []source.Personne
	lignesKV(body, func(kv map[string]string) {
		id := strings.TrimSpace(kv[cle])
		if id == "" {
			return
		}
		out = append(out, source.Personne{
			ExternalID: id,
			Nom:        strings.TrimSpace(kv["NOM"]),
			Prenom:     strings.TrimSpace(kv["PRENOM"]),
			Email:      strings.ToLower(strings.TrimSpace(kv["MEL"])),
		})
	})
	return out
}

func (s *Source) CoursNoms() (map[string]string, error) {
	body, err := fetchTexte(s.BaseURL + "?TYPE=cours_txt")
	if err != nil {
		return nil, fmt.Errorf("webdfd: cours_txt inaccessible: %w", err)
	}

	noms := make(map[string]string)
	lignesKV(body, func(kv map[string]string) {
		co := strings.TrimSpace(kv["CO"])
		nom := strings.TrimSpace(kv["NOM"])
		if co != "" && nom != "" {
			noms[co] = nom
		}
	})
	return noms, nil
}

func (s *Source) Planning(p0cle, debut, fin string) ([]source.Creneau, error) {
	url := fmt.Sprintf("%s?TYPE=planning_txt&DATEDEBUT=%s&DATEFIN=%s&TYPECLE=p0cleunik&VALCLE=%s",
		s.BaseURL, debut, fin, p0cle)

	body, err := fetchTexte(url)
	if err != nil {
		return nil, fmt.Errorf("webdfd: planning_txt P0=%s inaccessible: %w", p0cle, err)
	}

	var entries []source.Creneau
	lignesKV(body, func(kv map[string]string) {
		if kv["DATE"] == "" {
			return
		}
		entries = append(entries, source.Creneau{
			Plcle:  strings.TrimSpace(kv["PL"]),
			P0cle:  strings.TrimSpace(kv["P0CLE"]),
			Cocle:  strings.TrimSpace(kv["COCLE"]),
			Grcle:  strings.TrimSpace(kv["GRCLE"]),
			Prcle:  strings.TrimSpace(kv["PRCLE"]),
			Cours:  strings.TrimSpace(kv["COURS"]),
			Groupe: strings.TrimSpace(kv["GROUPE"]),
			Date:   strings.TrimSpace(kv["DATE"]),
			HD:     strings.TrimSpace(kv["HD"]),
			HF:     strings.TrimSpace(kv["HF"]),
			Salle:  strings.TrimSpace(kv["SALLE"]),
			Prof:   strings.TrimSpace(kv["PROF"]),
		})
	})
	return entries, nil
}

func (s *Source) MembresGroupe(grcle string) ([]string, string, error) {
	url := fmt.Sprintf("%s?TYPE=listegroupe_html&GRCLEUNIK=%s&MODE=10", s.ListeGroupeURL, grcle)
	resp, err := get(url)
	if err != nil {
		return nil, "", fmt.Errorf("webdfd: listegroupe GRCLE=%s inaccessible: %w", grcle, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	evs, groupeLabel, _ := parseListeGroupe(raw)
	return evs, groupeLabel, nil
}

// parseListeGroupe analyse le fichier listegroupe_html?MODE=10.
//
// Format observé (encodé Windows-1252, délimiteur tabulation) :
//
//	Promotion :     1A INFRES       Groupe :        INFRES  Matière :       ...
//	Num étudiant    Nom     Prénom  note
//	19475   AHMED ALI       Adoita
//
// Règles :
//   - Décoder en Windows-1252.
//   - Détecter le délimiteur sur les octets bruts (tabulation en priorité, sinon ';').
//   - Ignorer toutes les lignes dont le premier champ n'est pas numérique.
//   - Retourner uniquement le numéro d'étudiant (EV) ; la colonne note est ignorée.
//   - Retourner aussi les libellés Groupe et Promotion du préambule pour validation.
func parseListeGroupe(raw []byte) (evs []string, groupeLabel, promoLabel string) {
	// Détecter délimiteur sur les octets bruts.
	delim := detectDelimiter(raw)

	decoder := charmap.Windows1252.NewDecoder()
	decoded, _, _ := transform.Bytes(decoder, raw)

	for _, line := range strings.Split(string(decoded), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Extraction du préambule (première ligne non vide).
		if promoLabel == "" && strings.Contains(line, "Promotion") {
			promoLabel = extractLabel(line, "Promotion")
			groupeLabel = extractLabel(line, "Groupe")
			continue
		}
		fields := strings.Split(line, string(delim))
		if len(fields) == 0 {
			continue
		}
		first := strings.TrimSpace(fields[0])
		if !isNumeric(first) {
			continue
		}
		evs = append(evs, first)
	}
	return
}

// detectDelimiter inspecte les octets bruts pour choisir '\t' ou ';'.
func detectDelimiter(raw []byte) byte {
	if bytes.ContainsRune(raw, '\t') {
		return '\t'
	}
	return ';'
}

// extractLabel extrait la valeur après le mot-clé dans une ligne de préambule.
// Ex : "Promotion :     1A INFRES       Groupe :  INFRES" → "1A INFRES" pour "Promotion".
func extractLabel(line, keyword string) string {
	idx := strings.Index(line, keyword)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(keyword):]
	// Sauter le séparateur éventuel (':' et espaces).
	rest = strings.TrimLeft(rest, " :\t")
	// Arrêter au prochain mot-clé (Groupe, Matière, Contrôle…).
	for _, stop := range []string{"Groupe", "Matière", "Contrôle", "Coefficient", "\t"} {
		if i := strings.Index(rest, stop); i >= 0 {
			rest = rest[:i]
		}
	}
	return strings.TrimSpace(rest)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Garde-fou de compilation : Source doit satisfaire source.Source.
var _ source.Source = (*Source)(nil)
