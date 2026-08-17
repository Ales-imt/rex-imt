package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
)

// syncElevesGroupe écrit l'appartenance élève↔groupe pour les groupes dont la
// liste de membres a été récupérée pendant ce cycle.
//
// Pour chaque groupe :
//  1. Valide le libellé du groupe quand la source le fournit.
//  2. Résout chaque EV via migration.user_map ; ignore les EV inconnus.
//  3. Upsert eleve_groupe.
//  4. Retire les membres absents de la liste — sous condition, cf. plus bas.
//  5. Met à jour groupe.taille.
func syncElevesGroupe(ctx context.Context, q *Queries, c *collecte, ac anneeCourante) error {
	groupes, err := q.ListGroupeWebdfdIDs(ctx, ac.Annee)
	if err != nil {
		return err
	}

	var traites, ignores int
	for _, gr := range groupes {
		if !c.groupesOK[gr.ExternalID] {
			// Liste non récupérée ce cycle : le groupe garde son effectif.
			ignores++
			continue
		}
		if err := syncGroupe(ctx, q, gr.InternalID, gr.ExternalID, c); err != nil {
			return err
		}
		traites++
	}
	log.Printf("eleve_groupe: %d groupes traités, %d laissés intacts (liste non récupérée)", traites, ignores)
	return nil
}

func syncGroupe(ctx context.Context, q *Queries, internalID int64, grcle string, c *collecte) error {
	// Garde-fou : vérifier la cohérence du libellé quand la source le porte.
	if libelle := c.grcles[grcle].libelleAmont; libelle != "" {
		expectedLabel, err := q.GetGroupeLabel(ctx, internalID)
		if err == nil && expectedLabel != "" && !strings.EqualFold(libelle, expectedLabel) {
			log.Printf("eleve_groupe: GRCLE=%s: libellé flux %q ≠ DB %q (GRCLE périmé ?)", grcle, libelle, expectedLabel)
		}
	}

	idGroupe := int32(internalID)
	var seen []int32
	var skipped int
	for _, ev := range c.membres[grcle] {
		userID, err := q.GetUserBySource(ctx, GetUserBySourceParams{Source: source.Map, ExternalID: ev})
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("eleve_groupe: EV=%s introuvable dans user_map, ignoré (rechargez les élèves)", ev)
			skipped++
			continue
		}
		if err != nil {
			return fmt.Errorf("user_map: lookup EV=%s: %w", ev, err)
		}
		if err = q.UpsertEleveGroupe(ctx, UpsertEleveGroupeParams{NumEtudiant: userID, IDGroupe: idGroupe}); err != nil {
			return err
		}
		seen = append(seen, userID)
	}

	// Réconciliation — le geste destructeur de cette étape, et le seul du
	// cycle qui ne soit pas un simple marquage.
	//
	// DeleteEleveGroupeAbsents s'écrit `num_etudiant != ALL($2)`, un prédicat
	// VRAI pour tout le monde quand le tableau est vide : une liste vide vide
	// le groupe. Or l'amont sait rendre une liste vide sans erreur — une page
	// d'erreur cgiempt servie en HTTP 200 ne contient aucune ligne d'étudiant,
	// et rien ne la distingue d'un groupe réellement dépeuplé. La conséquence
	// se propage loin : par seance_effectif_resolu, l'effectif attendu de
	// toutes les séances à venir du groupe tomberait à zéro.
	//
	// On refuse donc de réconcilier sur une liste vide. Le prix est qu'un
	// groupe légitimement vidé en amont garde ses inscriptions jusqu'à une
	// intervention manuelle — sans commune mesure avec le coût de l'inverse.
	if len(seen) == 0 {
		log.Printf("eleve_groupe: ALERTE GRCLE=%s: aucun élève résolu sur %d EV annoncés — réconciliation refusée, effectif inchangé",
			grcle, len(c.membres[grcle]))
		return nil
	}

	if err := q.DeleteEleveGroupeAbsents(ctx, DeleteEleveGroupeAbsentsParams{
		IDGroupe: idGroupe,
		Column2:  seen,
	}); err != nil {
		return err
	}
	if err := q.UpdateGroupeTaille(ctx, UpdateGroupeTailleParams{Taille: int32(len(seen)), ID: internalID}); err != nil {
		return err
	}

	log.Printf("eleve_groupe: GRCLE=%s: %d élèves (%d ignorés)", grcle, len(seen), skipped)
	return nil
}
