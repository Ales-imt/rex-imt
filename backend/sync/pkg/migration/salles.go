package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// salleRef : ce que le résolveur rend pour un SACLE connu.
type salleRef struct {
	id  int64
	nom string
}

// resolveurSalle traduit le SACLE d'un créneau en salle du référentiel.
//
// UNE seule voie : la clé, via migration.salle_map. Le libellé du créneau n'est
// jamais utilisé pour retrouver une salle — il change en amont sans prévenir,
// et deux salles peuvent porter le même nom. Un rattachement faux est pire
// qu'un rattachement absent : il impute des heures d'occupation à la mauvaise
// salle, et rien ne le signale.
type resolveurSalle struct {
	parSacle map[string]salleRef
}

// resoudre rend l'identifiant de la salle ET son nom de référentiel.
//
// Le nom est rendu parce que seance.salle en dérive : la colonne texte cesse de
// recopier le libellé du créneau pour porter celui du référentiel, ce qui aligne
// l'écran d'occupation, les exports et les PDF de présence sur un seul
// vocabulaire. `nomCreneau` ne sert que de repli d'affichage quand la clé ne
// résout pas.
func (r resolveurSalle) resoudre(sacle, nomCreneau string) (pgtype.Int8, pgtype.Text) {
	if sacle != "" && sacle != "0" {
		if s, ok := r.parSacle[sacle]; ok {
			return pgtype.Int8{Int64: s.id, Valid: true},
				pgtype.Text{String: s.nom, Valid: true}
		}
	}
	nom := strings.TrimSpace(nomCreneau)
	return pgtype.Int8{}, pgtype.Text{String: nom, Valid: nom != ""}
}

// syncSalles upserte le référentiel des salles et rend de quoi rattacher les
// séances du cycle.
//
// Le résolveur est TOUJOURS construit depuis la base, jamais depuis la
// collecte : un cycle dont salles_txt n'a pas répondu doit continuer à
// rattacher les séances aux salles déjà connues. C'est la raison d'être du
// paramètre `ok` — il ne gouverne que l'écriture.
//
// Aucune salle n'est jamais supprimée : last_seen_at consigne la disparition,
// et une salle retirée de l'amont reste référencée par les séances passées.
func syncSalles(ctx context.Context, q *Queries, salles []source.Salle, ok bool) (resolveurSalle, error) {
	if ok {
		var crees, majs int
		for _, sa := range salles {
			// Capacité : 0 en amont = non renseignée, stockée NULL.
			capacite := pgtype.Int4{}
			if sa.Capacite > 0 {
				capacite = pgtype.Int4{Int32: int32(sa.Capacite), Valid: true}
			}
			typ := pgtype.Text{}
			if sa.Type != "" {
				typ = pgtype.Text{String: sa.Type, Valid: true}
			}

			salleID, err := q.GetSalleBySource(ctx, GetSalleBySourceParams{
				Source:     source.Map,
				ExternalID: sa.ExternalID,
			})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return resolveurSalle{}, fmt.Errorf("salle_map: lookup SA=%s: %w", sa.ExternalID, err)
			}
			if err == nil {
				if err = q.UpdateSalle(ctx, UpdateSalleParams{
					Name: sa.Nom, Capacite: capacite, Type: typ, ID: salleID,
				}); err != nil {
					return resolveurSalle{}, err
				}
				majs++
			} else {
				if salleID, err = q.CreateSalle(ctx, CreateSalleParams{
					Name: sa.Nom, Capacite: capacite, Type: typ,
				}); err != nil {
					return resolveurSalle{}, err
				}
				crees++
			}
			if err = q.UpsertSalleMap(ctx, UpsertSalleMapParams{
				InternalID: salleID,
				Source:     source.Map,
				ExternalID: sa.ExternalID,
			}); err != nil {
				return resolveurSalle{}, err
			}
		}
		log.Printf("salle: %d salles synchronisées (%d créées, %d mises à jour)", crees+majs, crees, majs)
	}

	return chargerResolveurSalle(ctx, q)
}

// chargerResolveurSalle lit l'index SACLE → salle depuis la base.
//
// Depuis la BASE et non depuis la collecte : le cycle doit continuer à
// rattacher les séances aux salles déjà connues même quand salles_txt n'a pas
// répondu. C'est aussi ce qui rend l'étape idempotente — deux cycles successifs
// sur le même amont produisent le même rattachement.
func chargerResolveurSalle(ctx context.Context, q *Queries) (resolveurSalle, error) {
	rows, err := q.ListSallesParSource(ctx, source.Map)
	if err != nil {
		return resolveurSalle{}, fmt.Errorf("salle: chargement du résolveur: %w", err)
	}

	r := resolveurSalle{parSacle: make(map[string]salleRef, len(rows))}
	for _, row := range rows {
		r.parSacle[row.ExternalID] = salleRef{id: row.ID, nom: row.Name}
	}
	log.Printf("salle: résolveur chargé, %d salles rattachables", len(r.parSacle))
	return r, nil
}
