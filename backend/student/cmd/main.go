package main

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/authentification"
	"back-rex-eleve/pkg/feedback"
	"back-rex-eleve/pkg/reponse"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Ces variables seront injectées au moment de la compilation
var (
	buildTime string
	version   string
)

func main() {

	// Affiche les informations de compilation
	log.Printf("Application version: %s", version)
	log.Printf("Compilation time: %s", buildTime)

	r := chi.NewRouter()
	r.Use(middleware.Logger) // Log HTTP requests

	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := services.LoadConfigYaml(configPath)
	if err != nil {
		log.Fatal("Erreur chargement config YAML :", err)
	}
	r.Use(services.MakeDatabaseMiddleware(&cfg.Database))
	auth.StartRefreshTokenCleanup(&cfg.Database)

	r.Use(services.FullLogRequest)

	// version api0
	r.Route("/api/v2", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			log.Println("Ping reçu api 1!")
			w.Write([]byte("pong"))
		})

		r.Route("/auth", func(r chi.Router) {
			auth.RoutesAuth(r, cfg, authentification.PostLdap)
		})
		role := []string{"ELEVE"}
		r.With(auth.Security(cfg.JWT, &role)).Route("/feedback", feedback.MakeRouteFeedBack(cfg.Age.PublicKey))
		r.With(auth.Security(cfg.JWT, &role)).Route("/reponse", reponse.MakeRouteReponse())

	})

	log.Printf("Serveur démarré sur le port %d (HTTP)", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", cfg.Server.Port),
		r,
	))
}
