package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

const (
	testServerURL = "https://159.31.247.10"
	testCACert    = "../x509/mtls_certs/ca.crt"
	testModel     = "mistral-small:latest"
	secretsEnv    = "../../.vscode/secrets.env"
)

func loadAPIKey(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("RACK_API_KEY"); v != "" {
		return v
	}
	f, err := os.Open(secretsEnv)
	if err != nil {
		t.Skipf("RACK_API_KEY absent et %s introuvable: %v", secretsEnv, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "RACK_API_KEY=") {
			return strings.TrimPrefix(line, "RACK_API_KEY=")
		}
	}
	t.Skip("RACK_API_KEY introuvable dans secrets.env")
	return ""
}

func getTestClient(t *testing.T) *http.Client {
	t.Helper()
	caCert, err := os.ReadFile(testCACert)
	if err != nil {
		t.Skipf("CA introuvable (%s), test ignoré: %v", testCACert, err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}
}

func TestListModels(t *testing.T) {
	client := getTestClient(t)
	apiKey := loadAPIKey(t)

	req, err := http.NewRequest(http.MethodGet, testServerURL+"/api/v0/ollama/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-API-KEY", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("requête échouée: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut inattendu: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	t.Logf("modèles: %s", body)
}

func TestSendPrompt(t *testing.T) {
	client := getTestClient(t)
	apiKey := loadAPIKey(t)

	payload, _ := json.Marshal(map[string]any{
		"model":  testModel,
		"prompt": "Explique moi le mTLS en deux phrases.",
		"stream": false,
	})

	req, err := http.NewRequest(http.MethodPost, testServerURL+"/api/v0/ollama/prompt", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("requête échouée: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut inattendu: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	t.Logf("réponse: %s", body)
}
