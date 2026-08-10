package main

import (
	"back-rex-admin/pkg/annee"
	"back-rex-admin/pkg/authentification"
	"back-rex-admin/pkg/bullettin"
	"back-rex-admin/pkg/curriculum"
	"back-rex-admin/pkg/evaluation"
	"back-rex-admin/pkg/feedback"
	ia "back-rex-admin/pkg/ia"
	"back-rex-admin/pkg/ia/ollama"
	"back-rex-admin/pkg/ia/rack"
	"back-rex-admin/pkg/ia/ragarenn"
	"back-rex-admin/pkg/justification"
	"back-rex-admin/pkg/migration"
	"back-rex-admin/pkg/moderation"
	"back-rex-admin/pkg/planning"
	"back-rex-admin/pkg/postit"
	"back-rex-admin/pkg/presence"
	"back-rex-admin/pkg/reponse"
	"back-rex-admin/pkg/reports"
	"back-rex-admin/pkg/rgpd"
	"back-rex-admin/pkg/user"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/health"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	// Embarque la base IANA des fuseaux dans le binaire : l'image runtime
	// (alpine sans paquet tzdata) ne fournit pas les zones, donc sans ceci
	// LoadLocation("Europe/Paris") retombe sur UTC et le planning est décalé.
	_ "time/tzdata"

	"github.com/go-chi/chi/v5"
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
	r.Use(services.LogOnError) // ne logge que les requêtes en erreur (>= 400)

	configPath := "/opt/rex-admin/conf/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := services.LoadConfigYaml(configPath)
	if err != nil {
		log.Fatal("Erreur chargement config YAML :", err)
	}

	configDir := filepath.Dir(configPath)
	if p := cfg.Presence.Timestamp.CaCertPath; p != "" && !filepath.IsAbs(p) {
		cfg.Presence.Timestamp.CaCertPath = filepath.Join(configDir, p)
	}
	r.Use(services.MakeDatabasePgMiddleware(&cfg.Database))
	auth.StartRefreshTokenCleanup(&cfg.Database)

	pg := services.NewPG(context.Background(), services.ToDBS(&cfg.Database))
	services.ConnectWithRetry("PostgreSQL", 5*time.Minute, func() error {
		return pg.Ping(context.Background())
	})
	var iaConnector ia.IAConnector
	switch cfg.IA.Provider {
	case "ollama":
		iaConnector = &ollama.Connector{
			BaseURL: cfg.Ollama.BaseURL,
			Model:   cfg.Ollama.Model,
		}
	case "rack":
		iaConnector = rack.New(cfg.Rack.BaseURL, cfg.Rack.APIKey, cfg.Rack.Model)
	case "ragarenn":
		iaConnector = ragarenn.New(cfg.RAGaRenn.BaseURL, cfg.RAGaRenn.APIKey, cfg.RAGaRenn.Model)
	default:
		log.Fatal("Ia inconnu")

	}

	migration.StartSync(context.Background(), migration.Config{
		PromosURL:      cfg.Webdfd.BaseURL + "?TYPE=promos_txt",
		CoursURL:       cfg.Webdfd.BaseURL + "?TYPE=cours_txt",
		PlanningURL:    cfg.Webdfd.BaseURL,
		ElevesURL:      cfg.Webdfd.BaseURL + "?TYPE=eleves_txt",
		ProfsURL:       cfg.Webdfd.BaseURL + "?TYPE=profs_txt",
		ListeGroupeURL: cfg.Webdfd.ListeGroupeURL,
	}, pg.Db)

	go feedback.ListenForNewFeedbacks(&cfg.Database, iaConnector)
	rgpd.StartPurge(&cfg.Database, &cfg.MariaDBConfig)

	// Auréga sert au CRUD utilisateur à dater les sorties, pour savoir si
	// l'horizon de conservation des pièces de présence est échu. Une connexion
	// indisponible n'est pas bloquante : sans date de sortie, aucun compte n'est
	// anonymisé (comportement fail-safe, cf. pkg/user/aurega.go).
	auregaDb, err := services.NewMariaDBConnection(cfg.MariaDBConfig)
	if err != nil {
		log.Printf("connexion Auréga indisponible, anonymisation sur demande admin désactivée: %v", err)
		auregaDb = nil
	}
	presence.StartAnchorScheduler(&cfg.Database, cfg.Presence)

	// permeet une analyse au lancement du programme.
	go feedback.ProcessPendingFeedbacks(&cfg.Database, iaConnector)
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			feedback.ProcessPendingFeedbacks(&cfg.Database, iaConnector)
		}
	}()

	// version api1
	r.Route("/api/v2", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			log.Println("Ping reçu api 1!")
			w.Write([]byte("pong"))
		})
		r.Get("/health", health.MakeHealthHandler(
			health.Checker{
				Name: "postgres",
				Check: func(ctx context.Context) error {
					return services.GetPgCtx(ctx).Ping(ctx)
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
		var iaBaseURL string
		switch cfg.IA.Provider {
		case "ollama":
			iaBaseURL = cfg.Ollama.BaseURL
		case "rack":
			iaBaseURL = cfg.Rack.BaseURL
		case "ragarenn":
			iaBaseURL = cfg.RAGaRenn.BaseURL
		}
		r.Get("/status", health.MakeStatusHandler(
			version, buildTime, startTime,
			health.HTTPChecker("ia_"+cfg.IA.Provider, iaBaseURL),
			health.LDAPChecker("ldap", cfg.LDAP.URL),
		))
		r.Route("/auth", func(r chi.Router) {
			auth.RoutesAuth(r, cfg, authentification.PostLdap)
		})

		adminOnly := []string{auth.RoleAdmin}
		adminAndGestionnaire := []string{auth.RoleAdmin, auth.RoleGestionnaire}
		// Garde de modération : ADMIN + rôle dédié MODERATEUR. L'inclusion de
		// GESTIONNAIRE reste à arbitrer avec le DPO (cf. résumé).
		moderationRoles := []string{auth.RoleAdmin, auth.RoleModerateur}

		r.With(auth.Security(cfg.JWT, &adminOnly)).
			Route("/user", func(r chi.Router) {
				user.RouteUtilisateur(r, cfg.LDAP, auregaDb)
			})
		r.With(auth.Security(cfg.JWT, &adminOnly)).
			Route("/annee", annee.RouteAnnee)
		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/feedback", feedback.RouteFeedback)
		r.With(auth.Security(cfg.JWT, &moderationRoles)).
			Route("/moderation", moderation.RouteModeration(cfg.Age.PublicKey))
		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/reports", reports.RouteReports)
		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/reponse", reponse.RouteReponse)
		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/postit", postit.RoutePostit)
		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/bullettin", func(r chi.Router) {
				bullettin.RouteBullettin(r, cfg.Bullettin)
			})
		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/evaluation", func(r chi.Router) {
				evaluation.RouteEvaluation(r, iaConnector)
			})
		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/presence", func(r chi.Router) {
				presence.RoutePresence(r, cfg.Presence)
			})

		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/justifications", justification.RouteJustification)

		r.With(auth.Security(cfg.JWT, &adminAndGestionnaire)).
			Route("/planning", planning.RoutePlanning)

		r.With(auth.Security(cfg.JWT, nil)).
			Route("/curriculum", curriculum.RouteCurriculum)
	})

	log.Printf("Serveur démarré sur le port %d (HTTP)", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(
		fmt.Sprintf(":%d", cfg.Server.Port),
		r,
	))
}
