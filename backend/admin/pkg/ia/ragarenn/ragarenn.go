package ragarenn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Connector appelle l'API RAGaRenn (compatible OpenAI) pour analyser un prompt.
type Connector struct {
	BaseURL string
	APIKey  string
	Model   string
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

// Analyze envoie le prompt à RAGaRenn et retourne la réponse textuelle.
func (c *Connector) Analyze(ctx context.Context, prompt string) (string, error) {
	time.Sleep(1 * time.Second)

	body, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("ragarenn: sérialisation requête: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ragarenn: création requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ragarenn: appel HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ragarenn: statut inattendu %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ragarenn: décodage réponse: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("ragarenn: aucun choix dans la réponse")
	}

	return result.Choices[0].Message.Content, nil
}
