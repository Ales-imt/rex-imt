package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// https://donchev.is/post/working-with-postgresql-in-go-using-pgx/

var PgCtxKey2 = &ContextKey{"pg entry"}

func ToDBS(cfg *DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)
}

func MakeDatabasePgMiddleware(cfg *DatabaseConfig) func(http.Handler) http.Handler {

	dsn := ToDBS(cfg)

	return func(next http.Handler) http.Handler {
		pg := NewPG(context.Background(), dsn)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), PgCtxKey2, pg)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// GetLogger récupère le logger enrichi depuis le contexte HTTP.
// Si aucun logger n'est trouvé dans le contexte, il retourne le logger global.
func GetPgCtx(ctx context.Context) *Postgres {
	if pgFromCtx, ok := ctx.Value(PgCtxKey2).(*Postgres); ok {
		return pgFromCtx
	}
	log.Fatal("GetPgCtx n'est pas du type Postgres")
	return nil // ne passera jamais ici, car sortira avant
}

type Postgres struct {
	Db *pgxpool.Pool
}

var (
	pgInstance *Postgres
	pgOnce     sync.Once
)

func NewPG(ctx context.Context, connString string) *Postgres {
	pgOnce.Do(func() {
		db, err := pgxpool.New(ctx, connString)

		if err != nil {
			log.Fatalf("unable to create connection pool: %v", err)
		}
		pgInstance = &Postgres{db}
	})

	return pgInstance
}

func (pg *Postgres) Ping(ctx context.Context) error {
	return pg.Db.Ping(ctx)
}

func (pg *Postgres) Close() {
	pg.Db.Close()
}
