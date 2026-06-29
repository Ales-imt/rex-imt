package migration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// SyncMatieres charge la liste des cours depuis webdfd (cours_txt) et upserte
// chaque matière dans public.matiere + migration.matiere_map.
// La période est dérivée du préfixe numérique du nom (ex : "9.2.1 NOM" → S9).
func SyncMatieres(ctx context.Context, url string, db *pgxpool.Pool) error {
	resp, err := webdfdGet(url)
	if err != nil {
		return fmt.Errorf("webdfd: cours_txt inaccessible: %w", err)
	}
	defer resp.Body.Close()

	decoder := charmap.Windows1252.NewDecoder()
	body, err := io.ReadAll(transform.NewReader(resp.Body, decoder))
	if err != nil {
		return err
	}

	_, anneeDebut, _ := schoolYear(time.Now())
	annee := int32(anneeDebut.Year())

	q := New(db)
	var count int
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "EOT" {
			continue
		}
		kv := parseKV(line)
		coStr := kv["CO"]
		nom := strings.TrimSpace(kv["NOM"])
		if coStr == "" || nom == "" {
			continue
		}
		if _, err = strconv.ParseInt(coStr, 10, 64); err != nil {
			continue
		}

		matiereID, err := q.GetMatiereBySource(ctx, GetMatiereBySourceParams{Source: "webdfd", ExternalID: coStr, Annee: annee})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("matiere_map: lookup webdfd CO=%s: %w", coStr, err)
		}
		if err == nil {
			if err = q.UpdateMatiereName(ctx, UpdateMatiereNameParams{Name: nom, ID: matiereID}); err != nil {
				return err
			}
		} else {
			matiereID, err = q.CreateMatiere(ctx, CreateMatiereParams{Name: nom, Annee: annee})
			if err != nil {
				return err
			}
		}
		if err = q.UpsertMatiereMap(ctx, UpsertMatiereMapParams{
			InternalID: matiereID,
			Source:     "webdfd",
			ExternalID: coStr,
			Annee:      annee,
		}); err != nil {
			return err
		}

		// Période : dérivée du nom si présent, et de la promo P0 si renseignée.
		if p0Str := kv["P0"]; p0Str != "" && p0Str != "0" {
			promoID, err := q.GetPromotionBySource(ctx, GetPromotionBySourceParams{Source: "webdfd", ExternalID: p0Str, Annee: annee})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("promotion_map: lookup pour periode P0=%s: %w", p0Str, err)
			}
			if err == nil {
				pname, ok := semesterFromName(nom)
				if !ok {
					pname = "INCONNU"
				}
				periodeID, err := q.UpsertPeriode(ctx, UpsertPeriodeParams{Name: pname, PromotionID: promoID, Annee: annee})
				if err != nil {
					return err
				}
				if err = q.UpdateMatierePeriode(ctx, UpdateMatierePeriodeParams{
					PeriodeID: pgtype.Int8{Int64: periodeID, Valid: true},
					ID:        matiereID,
				}); err != nil {
					return err
				}
			}
		}
		count++
	}
	log.Printf("matiere: %d matières synchronisées (annee=%d)", count, annee)
	return nil
}

// semesterFromName extrait "S<n>" d'un nom commençant par "<n>.<x>.<y>…".
// Ex : "9.2.1 PROJET" → "S9". Retourne ("INCONNU", false) si absent.
func semesterFromName(nom string) (string, bool) {
	idx := strings.IndexByte(nom, '.')
	if idx < 1 {
		return "INCONNU", false
	}
	prefix := nom[:idx]
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return "INCONNU", false
		}
	}
	n, err := strconv.Atoi(prefix)
	if err != nil || n <= 0 {
		return "INCONNU", false
	}
	return fmt.Sprintf("S%d", n), true
}
