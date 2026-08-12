package migration

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// seanceCandidate décrit une séance webdfd que la base juge périmée : sa
// correspondance seance_map n'a pas été rafraîchie pendant le cycle courant.
type seanceCandidate struct {
	id       int64  // public.seance.id
	plcle    string // identifiant amont du créneau (champ PL)
	promoID  int64  // 0 si la séance n'est rattachée à aucune promotion
	startsAt time.Time
	hasStart bool
}

// seancesPerimees décide quelles séances ont réellement disparu du planning
// amont, et doivent donc être annulées.
//
// La règle de fond est simple — un créneau qu'un cycle complet n'a pas revu
// n'existe plus — mais elle est DANGEREUSE appliquée telle quelle : dans
// SyncPlanning, une promo dont le fetch échoue est loguée puis ignorée. Sans
// précaution, une coupure réseau de quelques secondes annulerait tout le
// planning d'une promotion, et les feuilles de présence à venir avec.
//
// D'où quatre garde-fous, chacun protégeant d'une annulation prononcée sur la
// foi d'un flux incomplet :
//
//   - aucune promo récupérée → on n'annule rien du tout ;
//   - séance d'une promo non récupérée (ou sans promotion) → intacte ;
//   - séance hors de la plage réellement interrogée [debut, fin] → intacte,
//     car le cycle n'a rien demandé à son sujet ;
//   - créneau revu pendant ce cycle → intact, même si last_seen_at le fait
//     paraître périmé (décalage d'horloge entre l'application et la base).
func seancesPerimees(
	candidates []seanceCandidate,
	vus map[string]struct{},
	promosOK map[int64]bool,
	debut, fin time.Time,
) []int64 {
	if len(promosOK) == 0 {
		return nil
	}

	var ids []int64
	for _, c := range candidates {
		if _, revu := vus[c.plcle]; revu {
			continue
		}
		if !promosOK[c.promoID] {
			continue
		}
		if !c.hasStart || c.startsAt.Before(debut) || c.startsAt.After(fin) {
			continue
		}
		ids = append(ids, c.id)
	}
	return ids
}

// annulerSeancesDisparues confronte les séances webdfd non revues de ce cycle
// aux promos effectivement récupérées, puis marque les périmées.
// Retourne le nombre de séances annulées.
func annulerSeancesDisparues(
	ctx context.Context,
	q *Queries,
	cycleStart time.Time,
	vus map[string]struct{},
	promosOK map[int64]bool,
	debut, fin time.Time,
) (int64, error) {
	rows, err := q.ListSeancesWebdfdNonVues(ctx, pgtype.Timestamptz{Time: cycleStart, Valid: true})
	if err != nil {
		return 0, err
	}

	candidates := make([]seanceCandidate, 0, len(rows))
	for _, r := range rows {
		candidates = append(candidates, seanceCandidate{
			id:       r.InternalID,
			plcle:    r.ExternalID,
			promoID:  r.PromotionID.Int64, // 0 si NULL : aucune promo réussie ne matchera
			startsAt: r.StartsAt.Time,
			hasStart: r.StartsAt.Valid,
		})
	}

	ids := seancesPerimees(candidates, vus, promosOK, debut, fin)
	if len(ids) == 0 {
		return 0, nil
	}
	return q.MarkSeancesAnnulees(ctx, ids)
}
