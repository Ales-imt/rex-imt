package webdfd

import (
	"back-rex-eleve/pkg/programme"
	service_webfd "back-rex-eleve/pkg/service/webfd"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Connector récupère le planning depuis le serveur Cybema (webdfd.mines-ales.fr).
type Connector struct {
	ElevesURL   string
	PlanningURL string
	Eleves      service_webfd.ElevesCache
}

// Start initialise le cache des élèves en arrière-plan.
func (c *Connector) Start(ctx context.Context) {
	c.Eleves.Start(ctx, c.ElevesURL)
}

func formatHeure(h string) string {
	if len(h) == 4 {
		return h[:2] + ":" + h[2:]
	}
	return h
}

func formatDate(d string) string {
	if len(d) == 8 {
		return d[:4] + "-" + d[4:6] + "-" + d[6:]
	}
	return d
}

func parseKV(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
}

func (c *Connector) GetProgramme(_ context.Context, email, start, end string) ([]programme.Cours, error) {
	valcle, err := c.Eleves.Lookup(email)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s?TYPE=planning_txt&DATEDEBUT=%s&DATEFIN=%s&TYPECLE=evcleunik&VALCLE=%s",
		c.PlanningURL, start, end, valcle)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("webdfd: planning inaccessible: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cours []programme.Cours
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		if kv["DATE"] == "" {
			continue
		}
		cours = append(cours, programme.Cours{
			Date:  formatDate(kv["DATE"]),
			HD:    formatHeure(kv["HD"]),
			HF:    formatHeure(kv["HF"]),
			Cocle: kv["COCLE"],
			Cours: kv["COURS"],
			Salle: kv["SALLE"],
			Prof:  kv["PROF"],
			Promo: kv["PROMO"],
		})
	}
	if cours == nil {
		cours = []programme.Cours{}
	}
	return cours, nil
}
