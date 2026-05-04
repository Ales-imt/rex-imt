package mariadb

import (
	"back-rex-eleve/pkg/note"
	"back-rex-eleve/pkg/note/gen"
	"context"
	"database/sql"
	"regexp"
	"strings"
)

// Connector récupère les notes depuis la base MariaDB (Auréga).
type Connector struct {
	DB *sql.DB
}

var uePrefixRe = regexp.MustCompile(`^(\d+\.\d+)\.?([A-Za-z])?`)

func extractUEKey(name string) string {
	m := uePrefixRe.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	if m[2] != "" {
		return m[1] + strings.ToUpper(m[2])
	}
	return m[1]
}

func (c *Connector) GetNotes(ctx context.Context, email string) ([]note.Periode, error) {
	queries := gen.New(c.DB)

	rows, err := queries.GetNotesByEleve(ctx, sql.NullString{String: email, Valid: true})
	if err != nil {
		return nil, err
	}

	type periodeData struct {
		ueRows   []gen.GetNotesByEleveRow
		noteRows []gen.GetNotesByEleveRow
	}
	periodeMap := make(map[string]*periodeData)
	var periodeOrder []string

	for _, row := range rows {
		promo := row.Promo.String
		if _, ok := periodeMap[promo]; !ok {
			periodeMap[promo] = &periodeData{}
			periodeOrder = append(periodeOrder, promo)
		}
		if row.TypeExercice.String == "ECTS" {
			periodeMap[promo].ueRows = append(periodeMap[promo].ueRows, row)
		} else {
			periodeMap[promo].noteRows = append(periodeMap[promo].noteRows, row)
		}
	}

	var result []note.Periode
	for _, promo := range periodeOrder {
		data := periodeMap[promo]

		ueByKey := make(map[string]*note.UE)
		var ueOrder []string
		for _, row := range data.ueRows {
			key := extractUEKey(row.Matiere.String)
			ueByKey[key] = &note.UE{
				Nom:         row.Matiere.String,
				Score:       row.Noteobtenue.Float64,
				Coefficient: row.Coefficient.Float64,
			}
			ueOrder = append(ueOrder, key)
		}

		for _, row := range data.noteRows {
			key := extractUEKey(row.Matiere.String)
			if _, ok := ueByKey[key]; !ok {
				if _, exists := ueByKey[""]; !exists {
					ueByKey[""] = &note.UE{Nom: "Autres"}
					ueOrder = append(ueOrder, "")
				}
				key = ""
			}
			m := note.Matiere{
				Nom:         row.Matiere.String,
				Note:        row.Noteobtenue.Float64,
				Coefficient: row.Coefficient.Float64,
				Date:        row.DateExercice.Time.Format("2006-01-02"),
			}
			if row.Commentaire.Valid {
				s := row.Commentaire.String
				m.Commentaire = &s
			}
			ueByKey[key].Matieres = append(ueByKey[key].Matieres, m)
		}

		var ues []note.UE
		var totalWeight, weightedScore float64
		seen := make(map[string]bool)
		for _, key := range ueOrder {
			if seen[key] {
				continue
			}
			seen[key] = true
			ue := ueByKey[key]
			ues = append(ues, *ue)
			if ue.Coefficient > 0 {
				totalWeight += ue.Coefficient
				weightedScore += ue.Score * ue.Coefficient
			}
		}

		gpa := 0.0
		if totalWeight > 0 {
			gpa = weightedScore / totalWeight
		}

		result = append(result, note.Periode{
			Nom: promo,
			GPA: gpa,
			UEs: ues,
		})
	}

	return result, nil
}
