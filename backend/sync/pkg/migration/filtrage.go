package migration

import "log"

// filtre décide si une personne du référentiel amont mérite qu'on lui ouvre un
// compte.
//
// Le référentiel de cybema porte tout l'établissement ; le planning d'une année
// n'en fait venir qu'une fraction. Créer un compte pour les autres est une
// collecte de données personnelles sans finalité — et le stock ne se rattrape
// pas facilement une fois constitué.
//
// D'où la règle, volontairement asymétrique :
//
//	Une personne absente du planning n'est JAMAIS créée : ni compte, ni rôle,
//	ni ligne de map. Une personne déjà connue continue d'être mise à jour
//	comme avant.
//
// C'est un filtrage à l'écriture, pas une purge : rien n'est jamais supprimé
// ici. La base cesse de grossir, elle ne rétrécit pas. La réduction du stock
// existant relève du cycle de vie des comptes (backend/admin/pkg/rgpd), qui
// s'ancre sur la date de sortie et non sur la présence au planning.
type filtre struct {
	vus map[string]bool
	// actif vaut false quand la collecte n'a rien ramené sur quoi fonder une
	// décision. Le filtre s'efface alors devant le comportement d'avant : pas
	// d'information, pas de politique nouvelle — c'est le même principe qui
	// gouverne seancesPerimees.
	actif bool
}

// nouveauFiltre construit un filtre et journalise sa désactivation, sans quoi
// une dégradation de l'amont passerait inaperçue.
func nouveauFiltre(vus map[string]bool, actif bool, quoi string) filtre {
	if !actif {
		log.Printf("filtrage: %s non filtrés (aucune donnée de planning exploitable ce cycle)", quoi)
		return filtre{vus: vus, actif: false}
	}
	log.Printf("filtrage: %s — %d identifiants amont vus au planning", quoi, len(vus))
	return filtre{vus: vus, actif: true}
}

// retient rend vrai si l'on s'autorise à CRÉER quelque chose pour cet
// identifiant amont.
func (f filtre) retient(extID string) bool {
	return !f.actif || f.vus[extID]
}

// filtreProfs : un professeur compte s'il tient au moins un créneau du planning
// récupéré. Le filtre s'efface si aucune promo n'a répondu.
func (c *collecte) filtreProfs() filtre {
	return nouveauFiltre(c.prclesVus(), len(c.promosOK) > 0, "profs")
}

// filtreEleves : un élève compte s'il est inscrit dans au moins un groupe dont
// la liste a été lue. Le filtre s'efface si aucune liste n'a été obtenue.
func (c *collecte) filtreEleves() filtre {
	return nouveauFiltre(c.evsVus(), len(c.groupesOK) > 0, "élèves")
}
