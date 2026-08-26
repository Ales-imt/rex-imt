// Package bd sert le planning depuis public.seance, la table alimentée toutes
// les 2 h par la synchronisation de back-rex-admin.
//
// C'est la source nominale : elle évite un aller-retour vers Cybema à chaque
// consultation, et surtout elle sert exactement les mêmes séances que celles
// sur lesquelles se fait le pointage. Interroger webdfd en direct laissait
// coexister deux vérités — un cours pouvait s'afficher au planning sans exister
// dans public.seance, donc sans jamais pouvoir être pointé.
package bd

import (
	"back-rex-common/pkg/auth"
	"back-rex-eleve/pkg/programme"
	"back-rex-eleve/pkg/programme/bd/gen"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connector lit le planning en base. Aucun appel réseau, donc aucune source
// d'indisponibilité extérieure.
type Connector struct {
	DB *pgxpool.Pool
}

// parisLoc : les horaires sont stockés en timestamptz et doivent être rendus au
// front dans le fuseau des plannings, celui dans lequel la synchronisation les a
// écrits (cf. migration/planning.go).
var parisLoc = func() *time.Location {
	l, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.UTC
	}
	return l
}()

// ligne est la forme commune des trois requêtes : elles ne diffèrent que par
// leur filtre, jamais par leurs colonnes, pour que le DTO rendu soit identique
// quel que soit le rôle du demandeur.
type ligne struct {
	matiereID   int64
	matiereName string
	startsAt    pgtype.Timestamptz
	endsAt      pgtype.Timestamptz
	salle       string
	prof        string
	promo       string
	groupe      string
}

func (c *Connector) GetProgramme(ctx context.Context, d programme.Demandeur, debut, fin time.Time) ([]programme.Cours, error) {
	q := gen.New(c.DB)
	debutPg := pgtype.Timestamptz{Time: debut, Valid: true}
	finPg := pgtype.Timestamptz{Time: fin, Valid: true}

	var lignes []ligne
	// Ordre des rôles : gestionnaire > prof > élève. Un gestionnaire également
	// enregistré comme prof doit voir tout le planning, pas ses seuls cours.
	switch {
	case slices.Contains(d.Roles, auth.RoleGestionnaire):
		rows, err := q.ListProgrammeToutes(ctx, gen.ListProgrammeToutesParams{Debut: debutPg, Fin: finPg})
		if err != nil {
			return nil, fmt.Errorf("programme: lecture planning gestionnaire: %w", err)
		}
		for _, r := range rows {
			lignes = append(lignes, ligne{r.MatiereID, r.MatiereName, r.StartsAt, r.EndsAt, r.Salle, r.Prof, r.Promo, r.Groupe})
		}
	case slices.Contains(d.Roles, auth.RoleProf):
		rows, err := q.ListProgrammeProf(ctx, gen.ListProgrammeProfParams{
			ProfID: pgtype.Int4{Int32: d.UserID, Valid: true},
			Debut:  debutPg,
			Fin:    finPg,
		})
		if err != nil {
			return nil, fmt.Errorf("programme: lecture planning prof %d: %w", d.UserID, err)
		}
		for _, r := range rows {
			lignes = append(lignes, ligne{r.MatiereID, r.MatiereName, r.StartsAt, r.EndsAt, r.Salle, r.Prof, r.Promo, r.Groupe})
		}
	default:
		rows, err := q.ListProgrammeEleve(ctx, gen.ListProgrammeEleveParams{
			UserID: d.UserID,
			Debut:  debutPg,
			Fin:    finPg,
		})
		if err != nil {
			return nil, fmt.Errorf("programme: lecture planning élève %d: %w", d.UserID, err)
		}
		for _, r := range rows {
			lignes = append(lignes, ligne{r.MatiereID, r.MatiereName, r.StartsAt, r.EndsAt, r.Salle, r.Prof, r.Promo, r.Groupe})
		}
	}

	cours := make([]programme.Cours, 0, len(lignes))
	for _, l := range lignes {
		cours = append(cours, programme.Cours{
			Date:      formatJour(l.startsAt),
			HD:        formatHeure(l.startsAt),
			HF:        formatHeure(l.endsAt),
			MatiereID: l.matiereID,
			Cours:     l.matiereName,
			Salle:     l.salle,
			Prof:      l.prof,
			Promo:     l.promo,
			Groupe:    l.groupe,
		})
	}
	return cours, nil
}

func formatJour(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(parisLoc).Format("2006-01-02")
}

func formatHeure(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(parisLoc).Format("15:04")
}
