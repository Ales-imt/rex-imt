package service

import (
	"back-rex-eleve/pkg/service/gen"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const (
	matiereRefreshTTL = 2 * time.Hour
	matiereRetryDelay = 5 * time.Minute
)

func parseMatiereKV(line string) map[string]string {
	parts := strings.Split(strings.TrimSpace(line), ";")
	m := make(map[string]string, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[strings.TrimSpace(parts[i])] = strings.TrimSpace(parts[i+1])
	}
	return m
}

// semesterFromName extrait le numéro de semestre d'un nom de matière qui commence
// par un pattern de type "9.2.1 NOM COURS". Retourne (9, true) dans ce cas.
func semesterFromName(nom string) (int, bool) {
	idx := strings.IndexByte(nom, '.')
	if idx < 1 {
		return 0, false
	}
	prefix := nom[:idx]
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(prefix)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func syncMatieres(ctx context.Context, url string, db *pgxpool.Pool) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("webdfd: cours_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return err
	}

	q := gen.New(db)

	var count int
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseMatiereKV(line)
		coStr := kv["CO"]
		nom := strings.TrimSpace(kv["NOM"])
		if coStr == "" || nom == "" {
			continue
		}
		id, err := strconv.ParseInt(coStr, 10, 64)
		if err != nil {
			continue
		}

		var periodeID pgtype.Int8
		if p0Str := kv["P0"]; p0Str != "" {
			if promotionID, err := strconv.ParseInt(p0Str, 10, 64); err == nil {
				var pname string
				semNum, ok := semesterFromName(nom)
				if ok {
					pname = fmt.Sprintf("S%d", semNum)
				} else {
					pname = "INCONNU"
				}
				pid, err := q.UpsertPeriode(ctx, gen.UpsertPeriodeParams{
					Name:        pname,
					PromotionID: promotionID,
				})
				if err != nil {
					return fmt.Errorf("periode: upsert %q promotion %d: %w", pname, promotionID, err)
				}
				periodeID = pgtype.Int8{Int64: pid, Valid: true}
			}
		}

		if err = q.UpsertMatiere(ctx, gen.UpsertMatiereParams{
			ID:        id,
			Name:      nom,
			PeriodeID: periodeID,
		}); err != nil {
			return fmt.Errorf("matiere: upsert id=%d: %w", id, err)
		}
		count++
	}
	log.Printf("matiere: %d matières synchronisées", count)
	return nil
}
