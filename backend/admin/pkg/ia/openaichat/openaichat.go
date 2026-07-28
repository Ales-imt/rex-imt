// Package openaichat implémente le client commun aux fournisseurs IA exposant
// une API compatible OpenAI (endpoint /api/chat/completions). rack et ragarenn
// s'appuient dessus et ne portent que ce qui les distingue (transport HTTP,
// délai éventuel, préfixe d'erreur).
package openaichat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Connector appelle une API de chat compatible OpenAI et retourne la réponse textuelle.
type Connector struct {
	BaseURL string
	APIKey  string
	Model   string

	// Provider préfixe les messages d'erreur (ex: "rack", "ragarenn").
	Provider string
	// HTTPClient est utilisé pour l'appel ; http.DefaultClient si nil.
	HTTPClient *http.Client
	// Delay, si non nul, est attendu avant l'appel (ex: limitation de débit côté fournisseur).
	Delay time.Duration
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

// Analyze envoie le prompt au fournisseur et retourne la réponse textuelle.
func (c *Connector) Analyze(ctx context.Context, prompt string) (string, error) {
	if c.Delay > 0 {
		time.Sleep(c.Delay)
	}

	body, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("%s: sérialisation requête: %w", c.Provider, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("%s: création requête: %w", c.Provider, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: appel HTTP: %w", c.Provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: statut inattendu %d", c.Provider, resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("%s: décodage réponse: %w", c.Provider, err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("%s: aucun choix dans la réponse", c.Provider)
	}

	return result.Choices[0].Message.Content, nil
}
