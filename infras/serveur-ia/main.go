package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const AuthHeader = "X-API-KEY"

type config struct {
	Port      string `yaml:"port"`
	KeysFile  string `yaml:"keysFile"`
	OllamaURL string `yaml:"ollamaURL"`
}

var apiKeys = make(map[string]bool)

// Charger les clés API depuis le fichier spécifié
func loadKeys(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Erreur : Impossible d'ouvrir le fichier de clés '%s': %v", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key := strings.TrimSpace(scanner.Text())
		if key != "" && !strings.HasPrefix(key, "#") { // On ignore les lignes vides et les commentaires
			apiKeys[key] = true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Erreur lors de la lecture du fichier : %v", err)
	}

	fmt.Printf("✓ %d clés API chargées depuis : %s\n", len(apiKeys), filePath)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(AuthHeader)
		if !apiKeys[key] {
			log.Printf("Tentative d'accès refusée : IP=%s", r.RemoteAddr)
			http.Error(w, "Interdit: Clé API invalide", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func handleListModels(ollamaURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get(ollamaURL + "/api/tags")
		if err != nil {
			http.Error(w, "Ollama injoignable", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}
}

func handlePrompt(ollamaURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST requis", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		resp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(body))
		if err != nil {
			http.Error(w, "Erreur Ollama", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}
}

func loadConfig(path string) config {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Erreur lecture config '%s': %v", path, err)
	}
	var cfg config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("Erreur parsing config: %v", err)
	}
	return cfg
}

func main() {
	confPath := "conf.yaml"
	if len(os.Args) > 1 {
		confPath = os.Args[1]
	}
	cfg := loadConfig(confPath)

	loadKeys(cfg.KeysFile)

	http.HandleFunc("/api/v0/ollama/models", authMiddleware(handleListModels(cfg.OllamaURL)))
	http.HandleFunc("/api/v0/ollama/prompt", authMiddleware(handlePrompt(cfg.OllamaURL)))

	fmt.Printf("Serveur écoute sur %s (Mode Proxy Ollama)\n", cfg.Port)
	log.Fatal(http.ListenAndServe(cfg.Port, nil))
}
