package webdfd

import (
	"back-rex-eleve/pkg/programme"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connector récupère le planning depuis le serveur Cybema (webdfd.mines-ales.fr).
// La résolution email→EV se fait via migration.user_map (plus de cache mémoire).
type Connector struct {
	PlanningURL string
	DB          *pgxpool.Pool
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

// lookupWebdfdID retrouve l'identifiant webdfd (EV/evcleunik) d'un utilisateur
// via migration.user_map en joignant sur son email.
func (c *Connector) lookupWebdfdID(ctx context.Context, email string) (string, error) {
	var extID string
	err := c.DB.QueryRow(ctx,
		`SELECT um.external_id
		 FROM migration.user_map um
		 JOIN public."user" u ON u.id = um.internal_id
		 WHERE u.email = $1 AND um.source = 'webdfd'`,
		email).Scan(&extID)
	if err != nil {
		return "", fmt.Errorf("webdfd: étudiant non trouvé pour %s: %w", email, err)
	}
	return extID, nil
}

func (c *Connector) GetProgramme(ctx context.Context, email, start, end string) ([]programme.Cours, error) {
	valcle, err := c.lookupWebdfdID(ctx, email)
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
