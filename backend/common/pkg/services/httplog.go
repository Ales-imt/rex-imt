package services

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// LogOnError journalise une requête HTTP uniquement lorsqu'elle se solde par une
// erreur (statut >= 400). Les requêtes réussies ne produisent aucune ligne :
// logs propres en prod, tout en gardant la trace des échecs (4xx/5xx).
func LogOnError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(ww, r)

		if ww.Status() >= 400 {
			log.Printf("%d %s %s (%s) from %s",
				ww.Status(), r.Method, r.URL.Path,
				time.Since(start).Round(time.Millisecond), r.RemoteAddr)
		}
	})
}
