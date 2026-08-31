package migration

import (
	"back-rex-sync/pkg/source"
	"log"
	"time"
)

// collecte porte tout ce qu'un cycle a pu lire en amont, avant la moindre
// écriture. La séparation est structurelle, pas cosmétique :
//
//   - la phase d'écriture tient dans UNE transaction, et une transaction ne
//     doit jamais rester ouverte pendant un appel HTTP — cybema met plusieurs
//     secondes par promo, la connexion resterait idle in transaction tout ce
//     temps, verrous compris ;
//   - le filtrage à l'écriture (ne créer un compte que pour les personnes que
//     le planning fait effectivement venir) exige de connaître le planning
//     AVANT d'écrire profs et élèves.
//
// Aucun champ n'est renseigné depuis la base : la collecte n'interroge que la
// source. Les identifiants qu'elle porte sont donc tous des identifiants amont,
// et leur traduction en identifiants internes appartient à la phase d'écriture.
type collecte struct {
	promos    []source.Promo
	profs     []source.Personne
	eleves    []source.Personne
	coursNoms map[string]string

	// salles : référentiel amont des salles. sallesOK distingue « l'amont dit
	// qu'il n'y en a aucune » de « l'amont n'a pas répondu » — sans quoi une
	// panne de salles_txt effacerait tous les rattachements en base.
	salles   []source.Salle
	sallesOK bool

	// creneaux : PL → créneau. Le PL (PLCLEUNIK) est l'identifiant stable du
	// créneau amont, et l'unicité de seance_map. Un même PL vu sous deux promos
	// n'est retenu qu'une fois — cf. coclesMultiPromo pour la trace.
	creneaux map[string]source.Creneau
	cocles   map[string]cocleInfo
	grcles   map[string]grcleInfo

	// promosOK : P0 externes dont le planning a été récupéré SANS erreur. C'est
	// la seule liste sur laquelle il soit légitime d'annuler des séances ou de
	// conclure qu'un prof n'enseigne pas.
	promosOK map[string]bool

	// membres : GRCLE → EV des élèves du groupe.
	// groupesOK : GRCLE dont la liste a été récupérée ET jugée exploitable.
	membres   map[string][]string
	groupesOK map[string]bool

	// coclesMultiPromo : COCLE vus sous plusieurs P0 (cours mutualisés). Leur
	// période n'est rattachée qu'à une promotion, ce qui prive les autres
	// d'effectif attendu — on ne sait pas encore le corriger, on le signale.
	coclesMultiPromo map[string][]string

	// tracer active la mémorisation des créneaux écartés. Elle n'est allumée
	// que si un dump est configuré : garder chaque rejet coûte de la mémoire
	// proportionnelle au planning, pour une information dont un cycle nominal
	// n'a aucun usage.
	tracer bool
	rejets []rejet
	// bruts : nombre de créneaux rendus par la source, avant tout écartement.
	// Sert de dénominateur au bilan — c'est lui qu'on compare au nombre de
	// séances effectivement écrites.
	bruts int
}

// rejet consigne un créneau que la synchronisation n'a pas transformé en
// séance, et pourquoi. C'est le matériau du dump de diagnostic : sans lui, un
// créneau perdu l'est silencieusement, à l'un des cinq points d'écartement
// répartis entre la collecte et l'écriture.
type rejet struct {
	Raison  string         `json:"raison"`
	Promo   string         `json:"promo,omitempty"`
	Detail  string         `json:"detail,omitempty"`
	Creneau source.Creneau `json:"creneau"`
}

// Raisons d'écartement, du plus amont au plus aval.
const (
	rejetCocleVide       = "cocle_vide"        // créneau sans matière : rien à rattacher
	rejetPlcleVide       = "plcle_vide"        // sans PL, aucune identité stable côté seance_map
	rejetDoublonPL       = "doublon_pl"        // même PL rendu deux fois : le second écrase le premier
	rejetMatiereInconnue = "matiere_inconnue"  // COCLE écarté à l'étape des matières (nom vide)
	rejetHoraireIllisble = "horaire_illisible" // date ou heure de début non analysable
)

func (c *collecte) rejeter(raison, promo, detail string, e source.Creneau) {
	if !c.tracer {
		return
	}
	c.rejets = append(c.rejets, rejet{Raison: raison, Promo: promo, Detail: detail, Creneau: e})
}

type cocleInfo struct {
	displayName string
	promoExtID  string // P0 de la première promo où le COCLE a été vu
}

type grcleInfo struct {
	displayName string
	promoExtID  string
	// libelleAmont : libellé que la source associe au groupe dans sa liste de
	// membres, quand elle le fournit. Sert de garde-fou de cohérence contre le
	// nom déduit du planning — un écart signale un GRCLE périmé.
	libelleAmont string
}

// collecter lit tout ce que la source sait dire, sans rien écrire.
//
// `anneeOK` reprend le verdict de getAnneeCourante : sans année courante, le
// planning et les listes de groupes n'ont pas de plage à interroger, et la
// collecte se limite aux référentiels non datés (promos, profs, élèves).
//
// Seules les erreurs qui rendent le cycle entier ininterprétable sont
// remontées. Une promo ou un groupe inaccessible est journalisé et laissé de
// côté : il sort de promosOK / groupesOK, ce qui suffit à le soustraire à
// toutes les décisions destructrices en aval.
func collecter(src source.Source, ac anneeCourante, anneeOK, tracer bool) (*collecte, error) {
	c := &collecte{
		creneaux:         make(map[string]source.Creneau),
		cocles:           make(map[string]cocleInfo),
		grcles:           make(map[string]grcleInfo),
		promosOK:         make(map[string]bool),
		membres:          make(map[string][]string),
		groupesOK:        make(map[string]bool),
		coclesMultiPromo: make(map[string][]string),
		tracer:           tracer,
	}

	// Les sous-étapes sont journalisées au fur et à mesure : la collecte
	// concentre désormais TOUS les appels réseau, et sur un amont lent elle
	// peut occuper plusieurs minutes pendant lesquelles rien ne s'écrit en
	// base. Sans ces repères, un cycle en cours est indiscernable d'un service
	// bloqué — ou d'un cycle qui n'aurait jamais redémarré.
	log.Printf("collecte: référentiels (promotions, profs, élèves)…")
	var err error
	if c.promos, err = src.Promos(); err != nil {
		return nil, err
	}
	if c.profs, err = src.Profs(); err != nil {
		return nil, err
	}
	if c.eleves, err = src.Eleves(); err != nil {
		return nil, err
	}
	log.Printf("collecte: %d promotions, %d profs, %d élèves", len(c.promos), len(c.profs), len(c.eleves))

	// Une erreur ici n'interrompt pas le cycle : les salles enrichissent le
	// planning, elles ne le conditionnent pas. Même politique que pour une
	// promo inaccessible — on journalise et on laisse la base en l'état.
	if c.salles, err = src.Salles(); err != nil {
		log.Printf("salle: référentiel inaccessible: %v — rattachements conservés en l'état", err)
		c.salles = nil
	} else {
		c.sallesOK = true
	}

	if !anneeOK {
		return c, nil
	}
	if c.coursNoms, err = src.CoursNoms(); err != nil {
		return nil, err
	}

	c.collecterPlanning(src, ac)
	c.collecterMembres(src)
	return c, nil
}

// collecterPlanning interroge le planning de chaque promo du flux.
//
// La liste des promos vient de src.Promos() et non de migration.promotion_map :
// la collecte précède l'écriture, la map de ce cycle n'existe donc pas encore.
// Les deux listes coïncident de toute façon — promotion_map n'est peuplée que
// depuis ce flux — mais partir du flux évite qu'une promotion nouvelle attende
// un cycle de plus pour voir son planning lu.
func (c *collecte) collecterPlanning(src source.Source, ac anneeCourante) {
	debut := ac.Debut.Format("20060102")
	fin := ac.Fin.Format("20060102")
	log.Printf("collecte: planning de %d promotions sur %s → %s…", len(c.promos), debut, fin)

	for _, promo := range c.promos {
		entries, err := src.Planning(promo.ExternalID, debut, fin)
		if err != nil {
			log.Printf("planning: promo P0=%s inaccessible: %v — ignorée", promo.ExternalID, err)
			continue
		}
		c.promosOK[promo.ExternalID] = true

		c.bruts += len(entries)

		for _, e := range entries {
			if e.Cocle == "" || e.Cocle == "0" {
				// Sans matière, le créneau n'a rien à quoi se rattacher :
				// public.seance.matiere_id est NOT NULL.
				c.rejeter(rejetCocleVide, promo.ExternalID, "", e)
				continue
			}
			if vu, seen := c.cocles[e.Cocle]; !seen {
				c.cocles[e.Cocle] = cocleInfo{displayName: e.Cours, promoExtID: promo.ExternalID}
			} else if vu.promoExtID != promo.ExternalID {
				c.noterMultiPromo(e.Cocle, vu.promoExtID, promo.ExternalID)
			}

			if e.Plcle == "" {
				// Sans PL, aucune identité stable : le créneau serait recréé à
				// chaque cycle puis annulé, faute de correspondance seance_map.
				c.rejeter(rejetPlcleVide, promo.ExternalID, "", e)
			} else {
				if deja, doublon := c.creneaux[e.Plcle]; doublon {
					// Le second écrase le premier. Sur un cours mutualisé, la
					// promotion retenue dépend donc de l'ordre d'itération.
					c.rejeter(rejetDoublonPL, promo.ExternalID,
						"écrase le créneau déjà vu (COCLE="+deja.Cocle+", GRCLE="+deja.Grcle+")", deja)
				}
				c.creneaux[e.Plcle] = e
			}

			if e.Grcle != "" && e.Grcle != "0" {
				if _, seen := c.grcles[e.Grcle]; !seen {
					c.grcles[e.Grcle] = grcleInfo{displayName: e.Groupe, promoExtID: promo.ExternalID}
				}
			}
		}
	}
}

// noterMultiPromo enregistre qu'un COCLE apparaît sous plusieurs promotions.
func (c *collecte) noterMultiPromo(cocle string, promos ...string) {
	for _, p := range promos {
		if !contient(c.coclesMultiPromo[cocle], p) {
			c.coclesMultiPromo[cocle] = append(c.coclesMultiPromo[cocle], p)
		}
	}
}

func contient(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// collecterMembres lit la composition de chaque groupe rencontré dans le
// planning. Comme pour les promos, la liste des GRCLE vient des créneaux et non
// de migration.groupe_map : un groupe apparu ce cycle est ainsi peuplé
// immédiatement, et non au cycle suivant.
func (c *collecte) collecterMembres(src source.Source) {
	total := len(c.grcles)
	log.Printf("collecte: listes de %d groupes…", total)

	// C'est la sous-étape la plus longue : une requête par groupe, chacune
	// payant la latence réseau et le throttle de la source. Un point d'avancement
	// régulier évite de confondre « en cours » et « figé ».
	const pas = 50
	fait := 0

	for grcle := range c.grcles {
		fait++
		if fait%pas == 0 {
			log.Printf("collecte: %d/%d groupes lus", fait, total)
		}
		evs, libelle, err := src.MembresGroupe(grcle)
		if err != nil {
			log.Printf("eleve_groupe: GRCLE=%s inaccessible: %v — ignoré", grcle, err)
			continue
		}
		c.groupesOK[grcle] = true
		c.membres[grcle] = evs
		if libelle != "" {
			info := c.grcles[grcle]
			info.libelleAmont = libelle
			c.grcles[grcle] = info
		}
	}
}

// prclesVus rend les PRCLE portés par au moins un créneau du planning
// récupéré. C'est l'ensemble des professeurs que l'année fait effectivement
// venir devant des élèves.
func (c *collecte) prclesVus() map[string]bool {
	out := make(map[string]bool)
	for _, e := range c.creneaux {
		if e.Prcle != "" {
			out[e.Prcle] = true
		}
	}
	return out
}

// evsVus rend les EV des élèves inscrits dans au moins un groupe dont la liste
// a été récupérée. Le planning ne porte aucun identifiant d'élève : la seule
// voie est groupe → membres, et la couverture vaut donc ce que vaut le
// remplissage des GRCLE en amont.
func (c *collecte) evsVus() map[string]bool {
	out := make(map[string]bool)
	for grcle, evs := range c.membres {
		if !c.groupesOK[grcle] {
			continue
		}
		for _, ev := range evs {
			out[ev] = true
		}
	}
	return out
}

// journaliser trace ce que la collecte a ramené, avant toute écriture. C'est le
// seul endroit d'où l'on voit d'un coup d'œil si un cycle s'est appuyé sur un
// amont complet ou dégradé.
func (c *collecte) journaliser(debut time.Time) {
	log.Printf("collecte: %d promos (%d planning OK), %d créneaux, %d matières, %d groupes (%d listes OK), %d profs, %d élèves, %d salles — %s",
		len(c.promos), len(c.promosOK), len(c.creneaux), len(c.cocles),
		len(c.grcles), len(c.groupesOK), len(c.profs), len(c.eleves), len(c.salles),
		time.Since(debut).Round(time.Millisecond))

	for cocle, promos := range c.coclesMultiPromo {
		log.Printf("planning: ATTENTION COCLE=%s mutualisé entre les promos %v — sa période n'est rattachée qu'à la première, les autres n'auront pas d'effectif attendu",
			cocle, promos)
	}
}

// verifierContexte est appelé juste avant l'écriture : il rend une raison
// lisible quand la collecte est trop dégradée pour qu'on lui fasse confiance.
// Rien n'est bloqué ici — c'est chaque décision destructrice qui consultera
// promosOK / groupesOK — mais l'exploitant doit voir passer l'avertissement.
func (c *collecte) verifierContexte() {
	if len(c.promos) > 0 && len(c.promosOK) == 0 {
		log.Printf("collecte: ALERTE aucun planning récupéré sur %d promos — aucune séance ne sera annulée, aucun filtrage de profs appliqué", len(c.promos))
	}
	if len(c.grcles) > 0 && len(c.groupesOK) == 0 {
		log.Printf("collecte: ALERTE aucune liste de groupe récupérée sur %d groupes — aucun effectif ne sera réconcilié, aucun filtrage d'élèves appliqué", len(c.grcles))
	}
}
