package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b) // capture le body
	return lrw.ResponseWriter.Write(b)
}

func FullLogRequest(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("---- Requête entrante ----")
		fmt.Printf("Méthode: %s\n", r.Method)
		fmt.Printf("URL: %s\n", r.URL.String())
		fmt.Printf("Protocole: %s\n", r.Proto)
		fmt.Printf("Host: %s\n", r.Host)
		fmt.Printf("RemoteAddr: %s\n", r.RemoteAddr)
		fmt.Printf("Headers:\n")
		for name, values := range r.Header {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", name, value)
			}
		}
		if r.ContentLength > 0 {
			bodyBytes, _ := io.ReadAll(r.Body)
			fmt.Printf("Body: %s\n", string(bodyBytes))
			// Attention : il faut réinitialiser le body si tu veux le relire plus tard
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		// Wrap le ResponseWriter
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(lrw, r)

		// Log de la réponse
		fmt.Println("---- Réponse sortante ----")
		fmt.Printf("Status: %d\n", lrw.statusCode)
		for name, values := range lrw.Header() {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", name, value)
			}
		}
		fmt.Printf("Body: %s\n", lrw.body.String())
		fmt.Println("--------------------------")
	})
}
