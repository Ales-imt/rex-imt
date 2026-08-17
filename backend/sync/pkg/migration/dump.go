package migration

import (
	"back-rex-sync/pkg/source"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Le dump de diagnostic répond à une question précise et récurrente : « le
// planning amont contient N créneaux, la base en montre moins — où sont passés
// les autres ? »
//
// Y répondre depuis les logs est impossible : les créneaux sont écartés à cinq
// endroits distincts, répartis entre la collecte et l'écriture, et chacun le
// fait silencieusement parce qu'un écartement est le plus souvent légitime. Le
// dump rend l'ensemble inspectable d'un coup, créneau par créneau, avec la
// raison de chaque abandon.
//
// Il est désactivé par défaut : c'est un outil d'enquête, pas un journal.

// dumpCycle est le document écrit à chaque cycle quand migration.dumpPlanning
// est renseigné.
type dumpCycle struct {
	GenereLe string     `json:"genere_le"` // RFC3339 UTC
	Annee    dumpAnnee  `json:"annee"`
	Resume   dumpResume `json:"resume"`

	// Promos : une entrée par promotion du flux, y compris celles dont le
	// planning n'a pas répondu — c'est la première cause de créneaux manquants.
	Promos []dumpPromo `json:"promos"`

	// Creneaux : les créneaux retenus, ceux qui deviennent des séances. Triés
	// par date puis heure pour que deux dumps successifs se comparent au diff.
	Creneaux []source.Creneau `json:"creneaux"`

	// Rejets : les créneaux écartés, avec la raison. C'est le cœur du fichier.
	Rejets []rejet `json:"rejets"`
}

type dumpAnnee struct {
	Trouvee bool   `json:"trouvee"`
	Annee   int32  `json:"annee,omitempty"`
	Debut   string `json:"debut,omitempty"`
	Fin     string `json:"fin,omitempty"`
}

type dumpResume struct {
	CreneauxBruts   int            `json:"creneaux_bruts"`   // rendus par la source
	CreneauxRetenus int            `json:"creneaux_retenus"` // uniques, exploitables
	Rejets          int            `json:"rejets"`
	RejetsParRaison map[string]int `json:"rejets_par_raison"`
	PromosTotal     int            `json:"promos_total"`
	PromosOK        int            `json:"promos_ok"`
	GroupesTotal    int            `json:"groupes_total"`
	GroupesOK       int            `json:"groupes_ok"`
	Matieres        int            `json:"matieres"`
}

type dumpPromo struct {
	P0cle string `json:"p0cle"`
	Nom   string `json:"nom"`
	// PlanningOK à false = planning non récupéré. TOUS les créneaux de cette
	// promotion manquent alors au dump, et aucune de ses séances ne sera
	// annulée — c'est un trou, pas une suppression.
	PlanningOK bool `json:"planning_ok"`
	Creneaux   int  `json:"creneaux"`
}

// ecrireDump sérialise l'état de la collecte. Toute erreur est journalisée sans
// interrompre le cycle : un diagnostic qui ferait échouer la synchronisation
// qu'il observe serait pire que pas de diagnostic du tout.
func (c *collecte) ecrireDump(chemin string, ac anneeCourante, anneeOK bool) {
	if chemin == "" {
		return
	}

	doc := dumpCycle{
		GenereLe: time.Now().UTC().Format(time.RFC3339),
		Annee:    dumpAnnee{Trouvee: anneeOK},
		Rejets:   c.rejets,
	}
	if anneeOK {
		doc.Annee.Annee = ac.Annee
		doc.Annee.Debut = ac.Debut.Format("2006-01-02")
		doc.Annee.Fin = ac.Fin.Format("2006-01-02")
	}
	if doc.Rejets == nil {
		doc.Rejets = []rejet{}
	}

	// Créneaux triés : un dump doit se comparer au diff d'un cycle à l'autre,
	// or l'itération d'une map Go est volontairement aléatoire.
	doc.Creneaux = make([]source.Creneau, 0, len(c.creneaux))
	for _, e := range c.creneaux {
		doc.Creneaux = append(doc.Creneaux, e)
	}
	sort.Slice(doc.Creneaux, func(i, j int) bool {
		a, b := doc.Creneaux[i], doc.Creneaux[j]
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if a.HD != b.HD {
			return a.HD < b.HD
		}
		return a.Plcle < b.Plcle
	})
	sort.Slice(doc.Rejets, func(i, j int) bool {
		if doc.Rejets[i].Raison != doc.Rejets[j].Raison {
			return doc.Rejets[i].Raison < doc.Rejets[j].Raison
		}
		return doc.Rejets[i].Creneau.Plcle < doc.Rejets[j].Creneau.Plcle
	})

	// Créneaux par promotion, comptés sur les créneaux retenus.
	parPromo := make(map[string]int, len(c.promos))
	for _, e := range c.creneaux {
		parPromo[e.P0cle]++
	}
	doc.Promos = make([]dumpPromo, 0, len(c.promos))
	for _, p := range c.promos {
		doc.Promos = append(doc.Promos, dumpPromo{
			P0cle:      p.ExternalID,
			Nom:        p.Nom,
			PlanningOK: c.promosOK[p.ExternalID],
			Creneaux:   parPromo[p.ExternalID],
		})
	}
	sort.Slice(doc.Promos, func(i, j int) bool { return doc.Promos[i].P0cle < doc.Promos[j].P0cle })

	parRaison := make(map[string]int)
	for _, r := range c.rejets {
		parRaison[r.Raison]++
	}
	doc.Resume = dumpResume{
		CreneauxBruts:   c.bruts,
		CreneauxRetenus: len(c.creneaux),
		Rejets:          len(c.rejets),
		RejetsParRaison: parRaison,
		PromosTotal:     len(c.promos),
		PromosOK:        len(c.promosOK),
		GroupesTotal:    len(c.grcles),
		GroupesOK:       len(c.groupesOK),
		Matieres:        len(c.cocles),
	}

	blob, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Printf("dump: sérialisation impossible: %v", err)
		return
	}
	if err := ecrireAtomique(chemin, blob); err != nil {
		log.Printf("dump: écriture de %s impossible: %v", chemin, err)
		return
	}
	log.Printf("dump: %s — %d créneaux retenus sur %d bruts, %d rejets %v",
		chemin, doc.Resume.CreneauxRetenus, doc.Resume.CreneauxBruts, doc.Resume.Rejets, parRaison)
}

// ecrireAtomique écrit par fichier temporaire puis rename : un dump de
// plusieurs mégaoctets se lit souvent pendant qu'un cycle le réécrit, et un
// JSON tronqué ne se distingue pas d'un JSON incomplet.
//
// Le répertoire est créé au besoin. Attention toutefois : le service tourne
// sous un utilisateur non privilégié (`rex` dans l'image), qui ne peut pas
// créer d'arborescence sous une racine appartenant à root — /var/lib par
// exemple. Le chemin doit donc viser un emplacement que ce compte possède
// (/opt/rex-sync/… dans l'image) ou un volume monté en écriture.
func ecrireAtomique(chemin string, blob []byte) error {
	dir := filepath.Dir(chemin)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("répertoire %s inaccessible (le service tourne-t-il sous un compte qui peut y écrire ?): %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".dump-*.json")
	if err != nil {
		return err
	}
	nomTmp := tmp.Name()
	defer os.Remove(nomTmp) // sans effet après un rename réussi

	if _, err := tmp.Write(blob); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(nomTmp, 0o644); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return os.Rename(nomTmp, chemin)
}
