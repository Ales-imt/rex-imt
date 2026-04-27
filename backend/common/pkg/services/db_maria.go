package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var MariadbCtxKey = &ContextKey{"pg entry maria"}

// NewMariaDBConnection initialise une nouvelle connexion à une instance MariaDB.
// Elle configure également les paramètres du pool de connexions pour optimiser les performances.
func NewMariaDBConnection(cfg MariaDBConfig) (*sql.DB, error) {
	// Construction du DSN (Data Source Name)
	// parseTime=true est essentiel pour mapper les colonnes DATE/DATETIME vers time.Time en Go.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'ouverture de la base de données: %w", err)
	}

	// Configuration du pool de connexions
	// Ces valeurs devraient idéalement être extraites de la configuration environnementale.
	db.SetMaxOpenConns(25)                 // Nombre max de connexions ouvertes
	db.SetMaxIdleConns(25)                 // Nombre max de connexions inactives dans le pool
	db.SetConnMaxLifetime(5 * time.Minute) // Durée de vie maximale d'une connexion

	// Vérification immédiate de la validité de la connexion
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("impossible d'établir une connexion avec MariaDB: %w", err)
	}

	return db, nil
}

// MakeMariaDBMiddleware initialise la connexion et retourne un middleware pour injecter la DB dans le contexte Chi.
func MakeMariaDBMiddleware(cfg *MariaDBConfig) func(http.Handler) http.Handler {
	db, err := NewMariaDBConnection(*cfg)
	if err != nil {
		// On utilise panic car si la DB ne répond pas au démarrage, l'application ne peut pas fonctionner.
		panic(fmt.Sprintf("impossible de connecter MariaDB: %v", err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Injecte l'instance sql.DB dans le contexte de la requête
			ctx := context.WithValue(r.Context(), MariadbCtxKey, db)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetMariaDB est une fonction utilitaire pour récupérer la connexion MariaDB depuis le contexte.
func GetMariaDB(r *http.Request) *sql.DB {
	db, _ := r.Context().Value(MariadbCtxKey).(*sql.DB)
	return db
}
