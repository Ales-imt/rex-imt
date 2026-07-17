package main

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/health"
	"back-rex-common/pkg/presencetoken"
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/authentification"
	"back-rex-eleve/pkg/evaluation"
	"back-rex-eleve/pkg/feedback"
	"back-rex-eleve/pkg/note"
	mariadbnote "back-rex-eleve/pkg/note/mariadb"
	"back-rex-eleve/pkg/postit"
	"back-rex-eleve/pkg/programme"
	webdfdprog "back-rex-eleve/pkg/programme/webdfd"
	"back-rex-eleve/pkg/reponse"
	"back-rex-eleve/pkg/pointage"
	"back-rex-eleve/pkg/user"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Ces variables seront injectées au moment de la compilation
var (
	buildTime string
	version   string
)

func main() {

	startTime := time.Now()

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

	pg := services.NewPG(context.Background(), services.ToDBS(&cfg.Database))
	services.ConnectWithRetry("PostgreSQL", 5*time.Minute, func() error {
		return pg.Ping(context.Background())
	})

	var mariaDB *sql.DB
	services.ConnectWithRetry("MariaDB", 5*time.Minute, func() error {
		var err error
		mariaDB, err = services.NewMariaDBConnection(cfg.MariaDBConfig)
		return err
	})
	noteConnector := &mariadbnote.Connector{DB: mariaDB}
	progConnector := &webdfdprog.Connector{
		PlanningURL: "http://webdfd.mines-ales.fr/cybema/cgi-bin/cgiempt.exe",
		DB:          pg.Db,
	}

	// version api0
	r.Route("/api/v2", func(r chi.Router) {
		r.Get("/health", health.MakeHealthHandler(
			health.Checker{
				Name: "postgres",
				Check: func(ctx context.Context) error {
					return services.GetPgCtx(ctx).Ping(ctx)
				},
			},
			health.Checker{
				Name: "mariadb",
				Check: func(ctx context.Context) error {
					return mariaDB.PingContext(ctx)
				},
			},
		))
		r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"version":   version,
				"buildTime": buildTime,
			})
		})
		r.Get("/status", health.MakeStatusHandler(
			version, buildTime, startTime,
			health.HTTPChecker("webdfd", progConnector.PlanningURL),
			health.LDAPChecker("ldap", cfg.LDAP.URL),
		))

		r.Route("/auth", func(r chi.Router) {
			auth.RoutesAuth(r, cfg, authentification.PostLdap)
		})
		role := []string{auth.RoleEleve}
		r.With(auth.Security(cfg.JWT, &role)).Route("/feedback", feedback.MakeRouteFeedBack(cfg.Age.PublicKey))
		r.With(auth.Security(cfg.JWT, &role)).Route("/reponse", reponse.MakeRouteReponse())
		r.With(auth.Security(cfg.JWT, &role)).Route("/note", note.MakeRouteNote(noteConnector))
		r.With(auth.Security(cfg.JWT, &role)).Route("/programme", programme.MakeRouteProgramme(progConnector))
		r.With(auth.Security(cfg.JWT, &role)).Route("/evaluation", evaluation.MakeRouteEvaluation(cfg.Age.PublicKey))
		r.With(auth.Security(cfg.JWT, &role)).Route("/postit", postit.MakeRoutePostit())
		r.With(auth.Security(cfg.JWT, &role)).Route("/me", user.MakeRouteUser())
		presencetoken.SetSecret(cfg.Presence.TokenSecret)
		// /pointage mélange des routes élève (POST /) et prof/gestionnaire
		// (séances du jour, pilotage de séance) : authentification au montage,
		// contrôle de rôle par sous-route dans MakeRoutePointage.
		r.With(auth.Security(cfg.JWT, nil)).Route("/pointage", pointage.MakeRoutePointage())

	})

	log.Printf("Serveur démarré sur le port %d (HTTP)", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", cfg.Server.Port),
		r,
	))
}
