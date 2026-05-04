package webdfd

import (
	"back-rex-eleve/pkg/programme"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Connector récupère le planning depuis le serveur Cybema (webdfd.mines-ales.fr).
type Connector struct {
	ElevesURL      string
	PlanningURL    string
	CacheDuration  time.Duration
	elevesCache    elevesCache
}

type elevesCache struct {
	sync.RWMutex
	data      map[string]string
	expiresAt time.Time
}

// parseKV transforme "K1;V1;K2;V2;..." en map.
func parseKV(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
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

func (c *Connector) fetchElevesMap() (map[string]string, error) {
	resp, err := http.Get(c.ElevesURL)
	if err != nil {
		return nil, fmt.Errorf("webdfd: eleves_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		ev := kv["EV"]
		if ev == "" {
			continue
		}
		email := strings.ToLower(kv["MEL"])
		if email == "" {
			email = strings.ToLower(kv["EMAIL"])
		}
		if email == "" {
			continue
		}
		result[email] = ev
	}
	return result, nil
}

func (c *Connector) getValcle(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	ttl := c.CacheDuration
	if ttl == 0 {
		ttl = time.Hour
	}

	c.elevesCache.RLock()
	if time.Now().Before(c.elevesCache.expiresAt) {
		v, ok := c.elevesCache.data[email]
		c.elevesCache.RUnlock()
		if ok {
			return v, nil
		}
		return "", fmt.Errorf("webdfd: étudiant non trouvé pour %s", email)
	}
	c.elevesCache.RUnlock()

	c.elevesCache.Lock()
	defer c.elevesCache.Unlock()
	if time.Now().Before(c.elevesCache.expiresAt) {
		if v, ok := c.elevesCache.data[email]; ok {
			return v, nil
		}
		return "", fmt.Errorf("webdfd: étudiant non trouvé pour %s", email)
	}

	data, err := c.fetchElevesMap()
	if err != nil {
		return "", err
	}
	c.elevesCache.data = data
	c.elevesCache.expiresAt = time.Now().Add(ttl)

	v, ok := c.elevesCache.data[email]
	if !ok {
		return "", fmt.Errorf("webdfd: étudiant non trouvé pour %s", email)
	}
	return v, nil
}

func (c *Connector) GetProgramme(_ context.Context, email, start, end string) ([]programme.Cours, error) {
	valcle, err := c.getValcle(email)
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
