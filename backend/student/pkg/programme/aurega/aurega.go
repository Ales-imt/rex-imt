package aurega

import (
	"back-rex-common/pkg/auth"
	"back-rex-eleve/pkg/programme"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"
)

// jourFmt : format des paramètres de date attendu par l'API Aurega.
const jourFmt = "20060102"

// Connector récupère le planning depuis l'API Aurega.
type Connector struct {
	BaseURL string
	APIKey  string
}

func (c *Connector) GetProgramme(ctx context.Context, d programme.Demandeur, debut, fin time.Time) ([]programme.Cours, error) {
	// gestionnaire : toutes les séances de la période (email ignoré côté API).
	gestionnaire := slices.Contains(d.Roles, auth.RoleGestionnaire)
	// La borne haute de l'interface est exclusive, celle de l'API inclusive.
	url := fmt.Sprintf("%s/programme?email=%s&start=%s&end=%s&all=%t",
		c.BaseURL, d.Email, debut.Format(jourFmt), fin.AddDate(0, 0, -1).Format(jourFmt), gestionnaire)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("aurega: création requête: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aurega: appel HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aurega: statut inattendu %d", resp.StatusCode)
	}

	var cours []programme.Cours
	if err := json.NewDecoder(resp.Body).Decode(&cours); err != nil {
		return nil, fmt.Errorf("aurega: décodage réponse: %w", err)
	}
	return cours, nil
}
