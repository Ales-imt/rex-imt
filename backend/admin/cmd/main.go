package main

import (
	"back-rex-admin/pkg/authentification"
	"back-rex-admin/pkg/cohorte"
	"back-rex-admin/pkg/feedback"
	ia "back-rex-admin/pkg/ia"
	"back-rex-admin/pkg/ia/ollama"
	"back-rex-admin/pkg/ia/rack"
	"back-rex-admin/pkg/ia/ragarenn"
	"back-rex-admin/pkg/reports"
	"back-rex-admin/pkg/user"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	configDir := filepath.Dir(configPath)
	if !filepath.IsAbs(cfg.Rack.CaCertPath) {
		cfg.Rack.CaCertPath = filepath.Join(configDir, cfg.Rack.CaCertPath)
	}
	r.Use(services.MakeDatabaseMiddleware(&cfg.Database))
	auth.StartRefreshTokenCleanup(&cfg.Database)
	var iaConnector ia.IAConnector
	switch cfg.IA.Provider {
	case "ollama":
		iaConnector = &ollama.Connector{
			BaseURL: cfg.Ollama.BaseURL,
			Model:   cfg.Ollama.Model,
		}
	case "rack":
		iaConnector = &rack.Connector{
			BaseURL:    cfg.Rack.BaseURL,
			APIKey:     cfg.Rack.APIKey,
			Model:      cfg.Rack.Model,
			CaCertPath: cfg.Rack.CaCertPath,
		}
	case "ragarenn":
		iaConnector = &ragarenn.Connector{
			BaseURL: cfg.RAGaRenn.BaseURL,
			APIKey:  cfg.RAGaRenn.APIKey,
			Model:   cfg.RAGaRenn.Model,
		}
	default:
		log.Fatal("Ia inconnu")

	}
	go feedback.ProcessPendingFeedbacks(&cfg.Database, iaConnector)
	go feedback.ListenForNewFeedbacks(&cfg.Database, iaConnector)

	//r.Use(services.FullLogRequest)

	// version api1
	r.Route("/api/v2", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			log.Println("Ping reçu api 1!")
			w.Write([]byte("pong"))
		})
		r.Route("/auth", func(r chi.Router) {
			auth.RoutesAuth(r, cfg, authentification.PostLdap)
		})
		roles := []string{"ADMIN"}

		r.With(auth.Security(cfg.JWT, &roles)).
			Route("/user", func(r chi.Router) {
				user.RouteUtilisateur(r, cfg.LDAP)
			})
		r.With(auth.Security(cfg.JWT, &roles)).
			Route("/cohorte", func(r chi.Router) {
				cohorte.RouteCohorte(r, cfg.LDAP)
			})
		r.With(auth.Security(cfg.JWT, &roles)).
			Route("/feedback", feedback.RouteFeedback)
		r.With(auth.Security(cfg.JWT, &roles)).
			Route("/reports", reports.RouteReports)
	})

	log.Printf("Serveur démarré sur le port %d (HTTP)", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", cfg.Server.Port),
		r,
	))
}
