// Package rack construit le connecteur IA pour le rack personnel.
// Même logiciel que ragarenn (API compatible OpenAI, voir openaichat), mais
// le certificat HTTPS du rack est auto-signé : on désactive la vérification,
// comme "curl -k".
package rack

import (
	"crypto/tls"
	"net/http"

	"back-rex-admin/pkg/ia/openaichat"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

// New crée le connecteur pour le rack.
func New(baseURL, apiKey, model string) *openaichat.Connector {
	return &openaichat.Connector{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      model,
		Provider:   "rack",
		HTTPClient: httpClient,
	}
}
