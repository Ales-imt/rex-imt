package webdfd

import (
	"back-rex-common/pkg/auth"
	"back-rex-eleve/pkg/programme"
	"back-rex-eleve/pkg/programme/webdfd/gen"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
)

// jourFmt : format des paramètres DATEDEBUT/DATEFIN de cgiempt.exe.
const jourFmt = "20060102"

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

// lookupWebdfdID retrouve l'identifiant webdfd d'un utilisateur et le type de
// clé associé, en joignant sur son email :
//   - un étudiant est résolu via migration.user_map → TYPECLE=evcleunik ;
//   - un professeur est résolu via migration.prof_map → TYPECLE=prcleunik.
//
// La table user_map est consultée en priorité, puis prof_map en repli.
func (c *Connector) lookupWebdfdID(ctx context.Context, email string) (extID, typecle string, err error) {
	q := gen.New(c.DB)

	extID, err = q.GetWebdfdStudentExternalID(ctx, email)
	if err == nil {
		return extID, "evcleunik", nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("webdfd: résolution étudiant %s: %w", email, err)
	}

	extID, err = q.GetWebdfdProfExternalID(ctx, email)
	if err == nil {
		return extID, "prcleunik", nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", fmt.Errorf("webdfd: utilisateur non trouvé (ni étudiant ni prof) pour %s", email)
	}
	return "", "", fmt.Errorf("webdfd: résolution prof %s: %w", email, err)
}

func (c *Connector) GetProgramme(ctx context.Context, d programme.Demandeur, debut, fin time.Time) ([]programme.Cours, error) {
	start := debut.Format(jourFmt)
	// DATEFIN est INCLUSIVE côté Cybema, alors que la borne de l'interface est
	// exclusive : sans ce recul d'un jour, la journée suivant la plage demandée
	// serait ajoutée au planning.
	end := fin.AddDate(0, 0, -1).Format(jourFmt)

	// Gestionnaire : toutes les séances de la période, sans TYPECLE/VALCLE (pas
	// de filtrage par utilisateur, donc pas de résolution d'ID à faire).
	var url string
	if slices.Contains(d.Roles, auth.RoleGestionnaire) {
		url = fmt.Sprintf("%s?TYPE=planning_txt&DATEDEBUT=%s&DATEFIN=%s",
			c.PlanningURL, start, end)
	} else {
		valcle, typecle, err := c.lookupWebdfdID(ctx, d.Email)
		if err != nil {
			return nil, err
		}
		url = fmt.Sprintf("%s?TYPE=planning_txt&DATEDEBUT=%s&DATEFIN=%s&TYPECLE=%s&VALCLE=%s",
			c.PlanningURL, start, end, typecle, valcle)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("webdfd: planning inaccessible: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Cybema renvoie du texte en Windows-1252 (Latin-1 étendu) : sans conversion
	// vers UTF-8, les accents (é, è, à…) ressortent en � côté client.
	body, err = charmap.Windows1252.NewDecoder().Bytes(body)
	if err != nil {
		return nil, fmt.Errorf("webdfd: décodage Windows-1252: %w", err)
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
