package migration

import (
	"context"
	"os"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
)

// constSQL capture les requêtes générées par sqlc : `const nom = ` + "`" + `SQL` + "`" + `.
var constSQL = regexp.MustCompile("(?sm)^const (\\w+) = `(.*?)`$")

// TestToutesLesRequetesPreparent soumet chaque requête générée à PREPARE.
//
// sqlc valide la syntaxe et les noms de colonnes, mais PAS la résolution des
// types de paramètres, qui n'a lieu que dans le vrai analyseur de PostgreSQL.
// Un UpdateSeance parfaitement accepté par `sqlc generate` a ainsi échoué en
// production avec « could not determine data type of parameter $7 »
// (SQLSTATE 42P08) : un `@groupe_id IS NOT NULL` en RETURNING n'apporte aucune
// information de type, et le RETURNING est analysé AVANT la liste SET qui,
// elle, l'aurait typé.
//
// PREPARE est exactement l'étape qui manquait : il force cette résolution sans
// exécuter la moindre requête.
func TestToutesLesRequetesPreparent(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testDSN)
	if err != nil {
		t.Skipf("DB indisponible: %v", err)
	}
	defer conn.Close(ctx)

	// Le code sqlc est gitignoré mais toujours présent après `sqlc generate`,
	// que le Dockerfile impose avant tout build.
	src, err := os.ReadFile("query.sql.go")
	if err != nil {
		t.Fatalf("query.sql.go illisible (lancer `sqlc generate`): %v", err)
	}

	matches := constSQL.FindAllStringSubmatch(string(src), -1)
	if len(matches) == 0 {
		t.Fatal("aucune requête trouvée dans query.sql.go : le format généré a-t-il changé ?")
	}

	for _, m := range matches {
		nom, sql := m[1], m[2]
		t.Run(nom, func(t *testing.T) {
			// Chaque PREPARE dans sa propre transaction annulée : rien n'est
			// exécuté, et la session ne se charge pas de 47 statements.
			tx, err := conn.Begin(ctx)
			if err != nil {
				t.Fatalf("begin: %v", err)
			}
			defer tx.Rollback(ctx)

			if _, err := tx.Prepare(ctx, nom, sql); err != nil {
				t.Errorf("PREPARE échoue: %v", err)
			}
		})
	}
	t.Logf("%d requêtes vérifiées", len(matches))
}
