package services

import (
	"log"
	"time"
)

// ConnectWithRetry exécute connect() jusqu'au succès.
// En cas d'échec, attend interval puis réessaie indéfiniment.
func ConnectWithRetry(name string, interval time.Duration, connect func() error) {
	for {
		if err := connect(); err == nil {
			log.Printf("✅ %s connecté", name)
			return
		} else {
			log.Printf("⚠️  %s non joignable: %v — réessai dans %s", name, err, interval)
			time.Sleep(interval)
		}
	}
}
