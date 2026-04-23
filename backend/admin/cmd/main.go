package main

import (
	"back-rex-admin/pkg/authentification"
	"back-rex-admin/pkg/cohorte"
	"back-rex-admin/pkg/feedback"
	ia "back-rex-admin/pkg/ia"
	"back-rex-admin/pkg/ia/ollama"
	"back-rex-admin/pkg/ia/ragarenn"
	"back-rex-admin/pkg/reports"
	"back-rex-admin/pkg/user"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
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
	var iaConnector ia.IAConnector
	switch cfg.IA.Provider {
	case "ollama":
		iaConnector = &ollama.Connector{
			BaseURL: cfg.Ollama.BaseURL,
			Model:   cfg.Ollama.Model,
		}
	default:
		iaConnector = &ragarenn.Connector{
			BaseURL: cfg.RAGaRenn.BaseURL,
			APIKey:  cfg.RAGaRenn.APIKey,
			Model:   cfg.RAGaRenn.Model,
		}
	}
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

func newclient(cfg services.ServerConfig) (*http.Client, error) {

	CertPath := "/opt/rex-admin/cert"
	clientCertPath := CertPath + "/client.crt" // Le certificat du client (public)
	clientKeyPath := CertPath + "/client.key"  // La clé privée du client
	caCertPath := CertPath + "/ca.crt"         // Le certificat de l'autorité de certification (CA) du serveur

	// 1. Charger la clé et le certificat du client
	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		log.Fatalf("❌ Échec du chargement du certificat client: %v", err)
	}
	log.Println("✅ Certificat client chargé avec succès.")

	// 2. Préparer le pool de certificats CA du serveur (RootCAs)
	ca := x509.NewCertPool()
	caBytes, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("❌ Échec de la lecture du certificat CA serveur %q: %v", caCertPath, err)
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		log.Fatalf("❌ Échec du parsing du certificat CA serveur %q", caCertPath)
	}
	log.Println("✅ CA du serveur chargée pour la vérification.")

	// 3. Créer la configuration TLS
	tlsConfig := &tls.Config{
		ServerName:   "localhost",             // Doit correspondre au CN/SAN du 'server_cert.pem' du serveur de test
		Certificates: []tls.Certificate{cert}, // Certificat du client pour le mTLS
		RootCAs:      ca,                      // CA pour vérifier le certificat du serveur
		MinVersion:   tls.VersionTLS12,
	}

	// 4. Créer un transport HTTP personnalisé avec le DialContext corrigé
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		// Utilisation de net.Dialer pour la correction de l'erreur "http.Dialer est inconnu"
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	// 5. Créer le client HTTP
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return client, nil
}
