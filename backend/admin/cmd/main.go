package main

import (
	"back-rex-admin/pkg/authentification"
	"back-rex-admin/pkg/cohorte"
	"back-rex-admin/pkg/feedback"
	"back-rex-admin/pkg/ia/ollama"
	"back-rex-admin/pkg/user"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
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

	configPath := "/opt/rex-admin/conf/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := services.LoadConfigYaml(configPath)
	if err != nil {
		log.Fatal("Erreur chargement config YAML :", err)
	}
	r.Use(services.MakeDatabaseMiddleware(&cfg.Database))
	auth.StartRefreshTokenCleanup(&cfg.Database)
	iaConnector := &ollama.Connector{
		BaseURL: "http://localhost:11434",
		Model:   "mistral-nemo",
	}
	go feedback.ListenForNewFeedbacks(&cfg.Database, iaConnector)

	//	r.Use(services.FullLogRequest)
	getAccessToken := auth.GetAccessTokenByCookies

	// version api1
	r.Route("/api/v2", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			log.Println("Ping reçu api 1!")
			w.Write([]byte("pong"))
		})
		r.Route("/auth", func(r chi.Router) {
			auth.RoutesAuth(r, cfg, authentification.PostLdap, getAccessToken)
		})
		roles := []string{"admin"}

		r.With(auth.Security(cfg.JWT, getAccessToken, &roles)).
			Route("/user", func(r chi.Router) {
				user.RouteUtilisateur(r, cfg.LDAP)
			})
		r.With(auth.Security(cfg.JWT, getAccessToken, &roles)).
			Route("/cohorte", func(r chi.Router) {
				cohorte.RouteCohorte(r, cfg.LDAP)
			})
		r.With(auth.Security(cfg.JWT, getAccessToken, &roles)).
			Route("/feedback", feedback.RouteFeedback)
	})

	log.Printf("Serveur démarré sur le port %d (HTTP)", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", cfg.Server.Port),
		r,
	))
}
