package presence

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type SeanceCreneau struct {
	Date  string `json:"date"`
	HD    string `json:"hd"`
	HF    string `json:"hf"`
	Cours string `json:"cours"`
	Salle string `json:"salle"`
	Prof  string `json:"prof"`
	Promo string `json:"promo"`
}

func formatHeurePlanning(h string) string {
	if len(h) == 4 {
		return h[:2] + ":" + h[2:]
	}
	return h
}

func formatDatePlanning(d string) string {
	if len(d) == 8 {
		return d[:4] + "-" + d[4:6] + "-" + d[6:]
	}
	return d
}

func parseKVPlanning(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
}

// FetchSeancesMatiere récupère les créneaux d'une matière depuis webdfd.
// start et end sont optionnels (format YYYYMMDD).
func FetchSeancesMatiere(ctx context.Context, planningURL string, externalID int64, start, end string) ([]SeanceCreneau, error) {
	url := fmt.Sprintf("%s?TYPE=planning_txt&TYPECLE=cocleunik&VALCLE=%d", planningURL, externalID)
	if start != "" {
		url += "&DATEDEBUT=" + start
	}
	if end != "" {
		url += "&DATEFIN=" + end
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("planning: requête invalide: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("planning: inaccessible: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var seances []SeanceCreneau
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKVPlanning(line)
		if kv["DATE"] == "" {
			continue
		}
		seances = append(seances, SeanceCreneau{
			Date:  formatDatePlanning(kv["DATE"]),
			HD:    formatHeurePlanning(kv["HD"]),
			HF:    formatHeurePlanning(kv["HF"]),
			Cours: kv["COURS"],
			Salle: kv["SALLE"],
			Prof:  kv["PROF"],
			Promo: kv["PROMO"],
		})
	}
	if seances == nil {
		seances = []SeanceCreneau{}
	}
	return seances, nil
}
