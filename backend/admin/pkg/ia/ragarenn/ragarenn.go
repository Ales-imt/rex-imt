// Package ragarenn construit le connecteur IA pour RAGaRenn (API compatible
// OpenAI, voir openaichat).
package ragarenn

import (
	"time"

	"back-rex-admin/pkg/ia/openaichat"
)

// New crée le connecteur pour RAGaRenn.
func New(baseURL, apiKey, model string) *openaichat.Connector {
	return &openaichat.Connector{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Provider: "ragarenn",
		Delay:    1 * time.Second,
	}
}
