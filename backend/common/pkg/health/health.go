package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// Checker représente une vérification de santé nommée.
type Checker struct {
	Name  string
	Check func(ctx context.Context) error
}

// MakeHealthHandler retourne un handler HTTP qui exécute tous les checkers
// et répond 200 si tout est ok, 503 sinon.
func MakeHealthHandler(checkers ...Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		checks := make(map[string]string, len(checkers))
		allOk := true

		for _, c := range checkers {
			if err := c.Check(ctx); err != nil {
				checks[c.Name] = err.Error()
				allOk = false
			} else {
				checks[c.Name] = "ok"
			}
		}

		status := "ok"
		httpCode := http.StatusOK
		if !allOk {
			status = "error"
			httpCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpCode)
		json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"checks": checks,
		})
	}
}

// MakeStatusHandler retourne un handler qui affiche l'état de toutes les
// dépendances optionnelles ainsi que le contexte global du process.
// Répond toujours 200 — les erreurs sont informatives, pas bloquantes.
func MakeStatusHandler(version, buildTime string, start time.Time, checkers ...Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		deps := make(map[string]string, len(checkers))
		for _, c := range checkers {
			if err := c.Check(ctx); err != nil {
				deps[c.Name] = err.Error()
			} else {
				deps[c.Name] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"version":      version,
			"buildTime":    buildTime,
			"uptime":       time.Since(start).Round(time.Second).String(),
			"dependencies": deps,
		})
	}
}

// HTTPChecker retourne un Checker qui vérifie qu'une URL HTTP est joignable.
func HTTPChecker(name, url string) Checker {
	return Checker{
		Name: name,
		Check: func(ctx context.Context) error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		},
	}
}

// LDAPChecker retourne un Checker qui vérifie qu'un serveur LDAP est joignable.
func LDAPChecker(name, url string) Checker {
	return Checker{
		Name: name,
		Check: func(ctx context.Context) error {
			l, err := ldap.DialURL(url)
			if err != nil {
				return err
			}
			l.Close()
			return nil
		},
	}
}
