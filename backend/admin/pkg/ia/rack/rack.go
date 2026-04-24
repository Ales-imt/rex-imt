package rack

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Connector appelle l'API Ollama exposée sur le rack personnel via HTTPS.
type Connector struct {
	BaseURL    string
	APIKey     string
	Model      string
	CaCertPath string
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

// Analyze envoie le prompt au rack et retourne la réponse textuelle.
func (c *Connector) Analyze(ctx context.Context, prompt string) (string, error) {
	client, err := c.httpClient()
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(ollamaRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("rack: sérialisation requête: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v0/ollama/prompt", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("rack: création requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("rack: appel HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rack: statut inattendu %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("rack: décodage réponse: %w", err)
	}

	return result.Response, nil
}

func (c *Connector) httpClient() (*http.Client, error) {
	caCert, err := os.ReadFile(c.CaCertPath)
	if err != nil {
		return nil, fmt.Errorf("rack: lecture CA: %w", err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}, nil
}
