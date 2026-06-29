package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// SyncEleves charge la liste globale des élèves depuis webdfd (eleves_txt) et
// maintient la correspondance dans migration.user_map.
//
// Ordre de résolution pour chaque EV :
//  1. Chercher dans user_map (source='webdfd', external_id=EV) → mettre à jour les noms.
//  2. Sinon, chercher par email → créer uniquement la ligne user_map.
//  3. Sinon, créer l'utilisateur et la ligne user_map.
func SyncEleves(ctx context.Context, url string, db *pgxpool.Pool) error {
	resp, err := webdfdGet(url)
	if err != nil {
		return fmt.Errorf("webdfd: eleves_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return err
	}

	q := New(db)
	var count, created, linked int
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		ev := strings.TrimSpace(kv["EV"])
		if ev == "" {
			continue
		}
		nom := strings.TrimSpace(kv["NOM"])
		prenom := strings.TrimSpace(kv["PRENOM"])
		email := strings.ToLower(strings.TrimSpace(kv["MEL"]))
		if email == "" {
			log.Printf("eleve: EV=%s sans email, ignoré", ev)
			continue
		}

		// Étape 1 : lookup par EV dans user_map.
		userID, err := q.GetUserBySource(ctx, GetUserBySourceParams{Source: "webdfd", ExternalID: ev})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user_map: lookup EV=%s: %w", ev, err)
		}
		if err == nil {
			if err = q.UpdateUserNames(ctx, UpdateUserNamesParams{Name: nom, Surname: prenom, ID: userID}); err != nil {
				return err
			}
			if err = syncUserMap(ctx, q, userID, "webdfd", ev); err != nil {
				return err
			}
			count++
			continue
		}

		// Étape 2 : lookup par email.
		userID, err = q.GetUserByEmail(ctx, email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user: lookup email=%s: %w", email, err)
		}
		if err == nil {
			if err = syncUserMap(ctx, q, userID, "webdfd", ev); err != nil {
				return err
			}
			linked++
			count++
			continue
		}

		// Étape 3 : création de l'utilisateur.
		userID, err = createUserEleve(ctx, db, nom, prenom, email)
		if err != nil {
			return err
		}
		if err = syncUserMap(ctx, q, userID, "webdfd", ev); err != nil {
			return err
		}
		created++
		count++
	}
	log.Printf("eleve: %d élèves synchronisés (%d créés, %d rattachés par email)", count, created, linked)
	return nil
}
