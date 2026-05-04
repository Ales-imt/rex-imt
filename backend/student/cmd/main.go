package main

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/authentification"
	"back-rex-eleve/pkg/feedback"
	"back-rex-eleve/pkg/note"
	mariadbnote "back-rex-eleve/pkg/note/mariadb"
	"back-rex-eleve/pkg/programme"
	webdfdprog "back-rex-eleve/pkg/programme/webdfd"
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
	r.Use(services.MakeDatabasePgMiddleware(&cfg.Database))
	auth.StartRefreshTokenCleanup(&cfg.Database)

	mariaDB, err := services.NewMariaDBConnection(cfg.MariaDBConfig)
	if err != nil {
		log.Fatal("Erreur connexion MariaDB :", err)
	}
	noteConnector := &mariadbnote.Connector{DB: mariaDB}

	//r.Use(services.FullLogRequest)

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
		r.With(auth.Security(cfg.JWT, &role)).Route("/note", note.MakeRouteNote(noteConnector))
		progConnector := &webdfdprog.Connector{
				ElevesURL:   "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe?TYPE=eleves_txt",
				PlanningURL: "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe",
			}
			r.With(auth.Security(cfg.JWT, &role)).Route("/programme", programme.MakeRouteProgramme(progConnector))

	})

	log.Printf("Serveur démarré sur le port %d (HTTP)", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", cfg.Server.Port),
		r,
	))
}
