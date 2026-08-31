package migration

import (
	"back-rex-sync/pkg/source"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunSync exécute un cycle complet en trois temps : lire l'amont, décider, puis
// écrire d'un bloc.
//
//	COLLECTE  tous les appels réseau, aucune écriture, aucune transaction
//	FILTRAGE  en mémoire : qui le planning fait-il effectivement venir ?
//	ÉCRITURE  une seule transaction, aucun appel réseau
//
// La frontière entre les deux dernières phases et la première n'est pas
// négociable : garder une transaction ouverte pendant que cybema répond
// laisserait la connexion idle in transaction pendant des dizaines de secondes,
// verrous compris.
//
// Politique d'erreur, unique et énonçable en deux lignes :
//   - erreur de SOURCE (HTTP, contenu inexploitable) : tolérée. La promo ou le
//     groupe concerné sort de promosOK / groupesOK, ce qui le soustrait à
//     toutes les décisions destructrices en aval.
//   - erreur SQL : abandon et ROLLBACK. Le cycle est rejoué au tour suivant,
//     et la base n'aura jamais été vue à moitié synchronisée.
//
// Les étapes dépendant de l'année (promotions, planning, eleve_groupe) sont
// ignorées si aucune année de public.annee ne couvre la date du jour.
// dumpPlanning est le chemin du fichier de diagnostic, vide si désactivé. Il
// est passé par le scheduler au démarrage plutôt que porté par chaque appel :
// c'est un réglage de déploiement, constant pour la durée du service.
func RunSync(ctx context.Context, src source.Source, db *pgxpool.Pool, dumpPlanning string) error {
	debut := time.Now()
	// Marqueur d'ouverture de cycle. Il borne le journal par le bas : sans lui,
	// un cycle qui échoue pendant la collecte ne laisse qu'une trace d'erreur,
	// sans qu'on sache s'il a seulement démarré ni depuis quand il tournait.
	// L'heure est explicitement en UTC — le prefixe de log l'est aussi en
	// conteneur, mais par simple absence de /etc/localtime, ce sur quoi il
	// serait imprudent de compter pour dater un incident.
	log.Printf("sync: début du cycle (%s UTC)", debut.UTC().Format("2006-01-02 15:04:05"))

	// Lecture hors transaction : la plage [debut, fin] de l'année conditionne
	// les appels réseau de la collecte, il faut donc la connaître avant.
	ac, anneeOK, err := getAnneeCourante(ctx, New(db), time.Now())
	if err != nil {
		return err
	}
	if anneeOK {
		log.Printf("sync: année courante %d (%s → %s)",
			ac.Annee, ac.Debut.Format("2006-01-02"), ac.Fin.Format("2006-01-02"))
	} else {
		log.Printf("sync: aucune année ne correspond à la date du jour, étapes liées à l'année ignorées")
	}

	// --- Phase 1 : collecte ---
	c, err := collecter(src, ac, anneeOK, dumpPlanning != "")
	if err != nil {
		return err
	}
	c.journaliser(debut)
	c.verifierContexte()

	// Le dump est écrit en sortie de fonction, réussite ou échec : un cycle qui
	// casse à l'écriture est précisément celui dont on veut inspecter l'entrée.
	// Il est différé jusqu'ici pour inclure les rejets de la phase d'écriture,
	// que syncSeances ajoute à la même collecte.
	defer func() { c.ecrireDump(dumpPlanning, ac, anneeOK) }()

	// --- Phases 2 et 3 : filtrage et écriture ---
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("sync: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if err = ecrire(ctx, New(tx), c, ac, anneeOK); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("sync: commit: %w", err)
	}

	log.Printf("sync: cycle complet terminé en %s", time.Since(debut).Round(time.Millisecond))
	return nil
}

// ecrire déroule les étapes dans l'ordre imposé par leurs dépendances, toutes
// sur la même transaction.
//
//  0. salles       — rien (mais AVANT les séances : syncSeances y résout salle_id)
//  1. promotions   — rien
//  2. profs        — rien (mais AVANT les séances : syncSeances y résout prof_id)
//  3. matières     — promotions (période rattachée à une promo)
//  4. groupes      — promotions
//  5. séances      — matières, groupes, profs, salles
//  6. élèves       — rien
//  7. eleve_groupe — groupes, élèves
//  8. justifications — eleve_groupe (l'effectif attendu doit être celui du cycle)
func ecrire(ctx context.Context, q *Queries, c *collecte, ac anneeCourante, anneeOK bool) error {
	// L'horloge de référence du cycle vient de la BASE, pas de l'application :
	// last_seen_at est écrit par now(), et comparer deux horloges différentes
	// ferait passer des créneaux bien présents pour périmés. Dans une
	// transaction, now() vaut transaction_timestamp() — constante jusqu'au
	// COMMIT, et strictement égale au last_seen_at que ce cycle va écrire.
	cycleStart, err := q.Now(ctx)
	if err != nil {
		return fmt.Errorf("sync: horloge base: %w", err)
	}

	// 0. Salles. Hors année : le référentiel n'est pas annualisé.
	resSalle, err := syncSalles(ctx, q, c.salles, c.sallesOK)
	if err != nil {
		return err
	}

	// 1. Promotions.
	if anneeOK {
		if err := syncPromotions(ctx, q, c.promos, ac); err != nil {
			return err
		}
	}
	// 2. Profs, filtrés par le planning.
	if err := syncProfs(ctx, q, c); err != nil {
		return err
	}
	// 3, 4 et 5. Planning : matières et périodes, groupes, séances, annulations.
	var aRattacher []int64
	if anneeOK {
		if aRattacher, err = syncPlanning(ctx, q, c, ac, cycleStart, resSalle); err != nil {
			return err
		}
	}
	// 6. Élèves, filtrés par les listes de groupe.
	if err := syncEleves(ctx, q, c); err != nil {
		return err
	}
	// 7. Appartenance élève↔groupe.
	if anneeOK {
		if err := syncElevesGroupe(ctx, q, c, ac); err != nil {
			return err
		}
	}
	// 8. Couverture des excuses, une fois l'effectif attendu à jour.
	return rattacherJustifications(ctx, q, aRattacher)
}

// rattacherJustifications rejoue la couverture des excuses sur les seules
// séances qui en ont besoin : créées, rétablies, déplacées, ou dont l'effectif
// attendu a changé.
//
// justification_seance est matérialisée à la saisie de l'excuse ; une séance
// apparue ou déplacée APRÈS, dans une plage déjà couverte, ne serait sinon
// jamais excusée, et silencieusement.
//
// L'étape est volontairement la DERNIÈRE : la requête lit
// seance_effectif_resolu, qui remonte jusqu'à eleve_groupe. La placer avant la
// synchronisation des groupes — ce que faisait l'ancienne version, en
// l'appelant depuis syncSeances — la faisait travailler sur l'effectif du cycle
// PRÉCÉDENT.
//
// La restriction aux séances concernées n'est pas qu'une optimisation : cette
// jointure était jusqu'ici rejouée pour chaque séance de l'année à chaque
// cycle, soit le poste le plus coûteux de la synchronisation, et elle allonge
// désormais une transaction qui prend des verrous sur public.seance.
func rattacherJustifications(ctx context.Context, q *Queries, seances []int64) error {
	for _, id := range seances {
		if err := q.AttacherJustificationsSeance(ctx, id); err != nil {
			return fmt.Errorf("justification_seance: séance %d: %w", id, err)
		}
	}
	if len(seances) > 0 {
		log.Printf("planning: couverture des excuses revue sur %d séance(s)", len(seances))
	}
	return nil
}
