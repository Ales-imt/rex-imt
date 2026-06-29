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

// SyncProfs charge la liste des professeurs depuis webdfd (profs_txt) et
// maintient la correspondance dans migration.prof_map.
//
// Ordre de résolution pour chaque PRCLE :
//  1. Chercher dans prof_map (source='webdfd', external_id=PRCLE) → mettre à jour les noms.
//  2. Sinon, chercher par email → ajouter le rôle PROF + créer la ligne prof_map.
//  3. Sinon, créer l'utilisateur avec rôle PROF et la ligne prof_map.
func SyncProfs(ctx context.Context, url string, db *pgxpool.Pool) error {
	resp, err := webdfdGet(url)
	if err != nil {
		return fmt.Errorf("webdfd: profs_txt inaccessible: %w", err)
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
		prcle := strings.TrimSpace(kv["PR"])
		if prcle == "" {
			continue
		}
		nom := strings.TrimSpace(kv["NOM"])
		prenom := strings.TrimSpace(kv["PRENOM"])
		email := strings.ToLower(strings.TrimSpace(kv["MEL"]))
		if email == "" {
			log.Printf("prof: PRCLE=%s sans email, ignoré", prcle)
			continue
		}

		// Étape 1 : lookup par PRCLE dans prof_map.
		userID, err := q.GetProfBySource(ctx, GetProfBySourceParams{Source: "webdfd", ExternalID: prcle})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("prof_map: lookup PRCLE=%s: %w", prcle, err)
		}
		if err == nil {
			if err = q.UpdateUserNames(ctx, UpdateUserNamesParams{Name: nom, Surname: prenom, ID: userID}); err != nil {
				return err
			}
			if err = syncProfMap(ctx, q, userID, "webdfd", prcle); err != nil {
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
			if err = q.AddProfRole(ctx, userID); err != nil {
				return err
			}
			if err = syncProfMap(ctx, q, userID, "webdfd", prcle); err != nil {
				return err
			}
			linked++
			count++
			continue
		}

		// Étape 3 : création de l'utilisateur.
		userID, err = q.CreateUserProf(ctx, CreateUserProfParams{Email: email, Name: nom, Surname: prenom})
		if err != nil {
			return err
		}
		if err = syncProfMap(ctx, q, userID, "webdfd", prcle); err != nil {
			return err
		}
		created++
		count++
	}
	log.Printf("prof: %d profs synchronisés (%d créés, %d rattachés par email)", count, created, linked)
	return nil
}
