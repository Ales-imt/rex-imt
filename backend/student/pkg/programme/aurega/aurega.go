package aurega

import (
	"back-rex-eleve/pkg/programme"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Connector récupère le planning depuis l'API Aurega.
type Connector struct {
	BaseURL string
	APIKey  string
}

func (c *Connector) GetProgramme(ctx context.Context, email, start, end string) ([]programme.Cours, error) {
	url := fmt.Sprintf("%s/programme?email=%s&start=%s&end=%s", c.BaseURL, email, start, end)

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
