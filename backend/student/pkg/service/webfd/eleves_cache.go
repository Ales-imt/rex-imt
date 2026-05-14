package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	WebdfdCacheTTL   = 4 * time.Hour
	WebdfdRetryDelay = 5 * time.Minute
)

// ElevesCache maintient en mémoire la correspondance email → valcle webdfd.
// Appeler Start pour lancer le rafraîchissement automatique.
type ElevesCache struct {
	mu   sync.RWMutex
	data map[string]string
}

// Start lance la goroutine de fond qui charge et rafraîchit le cache depuis elevesURL.
// En cas d'erreur, elle réessaie après WebdfdRetryDelay ; en cas de succès,
// elle rafraîchit après WebdfdCacheTTL.
func (c *ElevesCache) Start(ctx context.Context, elevesURL string) {
	go func() {
		for {
			data, err := fetchElevesMap(elevesURL)
			if err != nil {
				log.Printf("webdfd: chargement cache élèves échoué: %v — nouvel essai dans %s", err, WebdfdRetryDelay)
				select {
				case <-time.After(WebdfdRetryDelay):
				case <-ctx.Done():
					return
				}
				continue
			}

			c.mu.Lock()
			c.data = data
			c.mu.Unlock()
			log.Printf("webdfd: cache élèves chargé (%d entrées)", len(data))

			select {
			case <-time.After(WebdfdCacheTTL):
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Lookup retourne le valcle associé à l'email, ou une erreur si introuvable.
func (c *ElevesCache) Lookup(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil {
		return "", fmt.Errorf("webdfd: cache élèves non encore initialisé")
	}
	v, ok := c.data[email]
	if !ok {
		return "", fmt.Errorf("webdfd: étudiant non trouvé pour %s", email)
	}
	return v, nil
}

func fetchElevesMap(elevesURL string) (map[string]string, error) {
	resp, err := http.Get(elevesURL)
	if err != nil {
		return nil, fmt.Errorf("webdfd: eleves_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseWebdfdKV(line)
		ev := kv["EV"]
		if ev == "" {
			continue
		}
		email := strings.ToLower(kv["MEL"])
		if email == "" {
			email = strings.ToLower(kv["EMAIL"])
		}
		if email == "" {
			continue
		}
		result[email] = ev
	}
	return result, nil
}

func parseWebdfdKV(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
}
