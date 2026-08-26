package hfsql

import (
	"back-rex-sync/pkg/source"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Source lit un export JSON produit par ce paquet à partir d'une sauvegarde
// HFSQL de cybema, et le sert comme référentiel à pkg/source.
//
// Tout est chargé et indexé au montage : un export est un instantané cohérent
// de quelques Mo, et le relire à chaque étape de la synchronisation reviendrait
// à décoder six fois les mêmes 18 000 créneaux. Une donnée manquante ou
// illisible se signale ici, avant que la synchronisation ne commence.
type Source struct {
	dir string

	promos []source.Promo
	profs  []source.Personne
	eleves []source.Personne

	coursNoms map[string]string // COCLEUNIK -> nom complet
	salles    map[string]string // SACLEUNIK -> nom
	groupes   map[string]string // GRCLEUNIK -> nom
	profsNoms map[string]string // PRCLEUNIK -> nom d'affichage
	planning  map[string][]source.Creneau
	membres   map[string][]string // GRCLEUNIK -> EVCLEUNIK
}

// clé HFSQL : les CLEUNIK sont exportés en nombre, les tables de correspondance
// les manipulent en chaîne (comme les identifiants webdfd).
type cleunik int64

func (c cleunik) String() string { return strconv.FormatInt(int64(c), 10) }

// Structures de décodage. Les champs absents d'un fichier restent à zéro : on
// ne décode que ce que la synchronisation consomme.
type (
	hfPromo struct {
		P0CLEUNIK cleunik
		Nom       string `json:"nom"`
	}
	hfPersonne struct {
		PRCLEUNIK cleunik
		EVCLEUNIK cleunik
		Det       string `json:"det"` // civilité ("M.", "Mme")
		Nom       string `json:"nom"`
		Prenom    string `json:"prenom"`
		Mel       string `json:"mel"`
	}
	hfCours struct {
		COCLEUNIK cleunik
		Nom       string `json:"nom"`
	}
	hfGroupe struct {
		GRCLEUNIK cleunik
		Nom       string `json:"nom"`
	}
	hfSalle struct {
		SACLEUNIK cleunik
		Nom       string `json:"nom"`
	}
	hfEleGro struct {
		GRCLEUNIK cleunik
		EVCLEUNIK cleunik
	}
	hfPlanning struct {
		PLCLEUNIK  cleunik
		P0CLEUNIK  cleunik
		COCLEUNIK  cleunik
		GRCLEUNIK  cleunik
		PRCLEUNIK  cleunik
		SACLEUNIK  cleunik
		Jour       string `json:"jour"`
		Heuredebut string `json:"heuredebut"`
		Heurefin   string `json:"heurefin"`
		Note       string `json:"note"`
	}
)

// NewSource monte la source sur un répertoire json d'export.
func NewSource(dir string) (*Source, error) {
	s := &Source{dir: dir}

	var promos []hfPromo
	var profs, eleves []hfPersonne
	var cours []hfCours
	var groupes []hfGroupe
	var salles []hfSalle
	var eleGro []hfEleGro
	var creneaux []hfPlanning

	// Le nom du fichier fait foi, pas celui de la clé racine : l'export nomme
	// ses tables tantôt "promos", tantôt "Eleves", tantôt "COURS".
	for _, t := range []struct {
		fichier string
		dest    any
	}{
		{"promos.json", &promos},
		{"profs.json", &profs},
		{"eleves.json", &eleves},
		{"cours.json", &cours},
		{"Groupes.json", &groupes},
		{"salles.json", &salles},
		{"eleGro.json", &eleGro},
		{"planning.json", &creneaux},
	} {
		if err := lireTable(dir, t.fichier, t.dest); err != nil {
			return nil, err
		}
	}

	for _, p := range promos {
		if nom := champ(p.Nom); nom != "" {
			s.promos = append(s.promos, source.Promo{ExternalID: p.P0CLEUNIK.String(), Nom: nom})
		}
	}
	s.profsNoms = make(map[string]string, len(profs))
	for _, p := range profs {
		s.profs = append(s.profs, source.Personne{
			ExternalID: p.PRCLEUNIK.String(),
			Nom:        champ(p.Nom),
			Prenom:     champ(p.Prenom),
			Email:      strings.ToLower(champ(p.Mel)),
		})
		// Nom d'affichage du planning : civilité + NOM, sans prénom. C'est le
		// format que webdfd sert dans son champ PROF (« M. VIELJUS »), vérifié
		// sur le flux planning_txt ; s'en écarter changerait l'affichage des
		// séances au basculement de source, et pour les séances déjà en base.
		s.profsNoms[p.PRCLEUNIK.String()] = strings.TrimSpace(champ(p.Det) + " " + champ(p.Nom))
	}
	for _, e := range eleves {
		s.eleves = append(s.eleves, source.Personne{
			ExternalID: e.EVCLEUNIK.String(),
			Nom:        champ(e.Nom),
			Prenom:     champ(e.Prenom),
			Email:      strings.ToLower(champ(e.Mel)),
		})
	}

	s.coursNoms = make(map[string]string, len(cours))
	for _, c := range cours {
		if nom := champ(c.Nom); nom != "" {
			s.coursNoms[c.COCLEUNIK.String()] = nom
		}
	}
	s.groupes = make(map[string]string, len(groupes))
	for _, g := range groupes {
		s.groupes[g.GRCLEUNIK.String()] = champ(g.Nom)
	}
	s.salles = make(map[string]string, len(salles))
	for _, sa := range salles {
		s.salles[sa.SACLEUNIK.String()] = champ(sa.Nom)
	}

	s.membres = make(map[string][]string)
	for _, eg := range eleGro {
		gr := eg.GRCLEUNIK.String()
		s.membres[gr] = append(s.membres[gr], eg.EVCLEUNIK.String())
	}

	// L'export livre le planning de toutes les promotions et de toutes les
	// années d'un bloc ; l'indexer par promotion évite de le reparcourir en
	// entier à chaque appel de Planning.
	s.planning = make(map[string][]source.Creneau)
	for _, c := range creneaux {
		p0 := c.P0CLEUNIK.String()
		s.planning[p0] = append(s.planning[p0], s.creneau(c))
	}

	return s, nil
}

// creneau convertit une ligne de planning.json en source.Creneau.
//
// L'export ne porte que des clés là où webdfd livrait aussi les libellés :
// salle, professeur, matière et groupe sont résolus ici par jointure sur les
// autres tables de l'export.
func (s *Source) creneau(c hfPlanning) source.Creneau {
	return source.Creneau{
		Plcle:  c.PLCLEUNIK.String(),
		P0cle:  c.P0CLEUNIK.String(),
		Cocle:  c.COCLEUNIK.String(),
		Grcle:  c.GRCLEUNIK.String(),
		Prcle:  c.PRCLEUNIK.String(),
		Cours:  s.coursNoms[c.COCLEUNIK.String()],
		Groupe: s.groupes[c.GRCLEUNIK.String()],
		Date:   champ(c.Jour),
		HD:     champ(c.Heuredebut),
		HF:     champ(c.Heurefin),
		Salle:  s.salles[c.SACLEUNIK.String()],
		Prof:   s.profsNoms[c.PRCLEUNIK.String()],
		Note:   champ(c.Note),
	}
}

func (s *Source) Promos() ([]source.Promo, error)       { return s.promos, nil }
func (s *Source) Profs() ([]source.Personne, error)     { return s.profs, nil }
func (s *Source) Eleves() ([]source.Personne, error)    { return s.eleves, nil }
func (s *Source) CoursNoms() (map[string]string, error) { return s.coursNoms, nil }

// Planning filtre l'instantané sur la promotion et la plage demandées. webdfd
// délègue ce filtrage au serveur ; ici il se fait en mémoire, sur des dates au
// format AAAAMMJJ dont l'ordre lexicographique est l'ordre chronologique.
func (s *Source) Planning(p0cle, debut, fin string) ([]source.Creneau, error) {
	var out []source.Creneau
	for _, e := range s.planning[p0cle] {
		if e.Date == "" || e.Date < debut || e.Date > fin {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// MembresGroupe rend les élèves du groupe. Le libellé est vide : l'export ne
// porte pas le préambule que webdfd sert avec sa liste, donc pas de garde-fou
// de cohérence à ce niveau — l'instantané est cohérent par construction.
func (s *Source) MembresGroupe(grcle string) ([]string, string, error) {
	return s.membres[grcle], "", nil
}

// lireTable décode le tableau unique contenu dans un fichier de l'export.
//
// Chaque fichier tient dans un objet à une seule clé, dont le nom varie en
// casse et en orthographe d'une table à l'autre ("promos", "Eleves", "COURS",
// "eleGro"). Plutôt que d'entretenir cette liste, on prend l'unique valeur.
func lireTable(dir, fichier string, dest any) error {
	chemin := filepath.Join(dir, fichier)
	raw, err := os.ReadFile(chemin)
	if err != nil {
		return fmt.Errorf("export hfsql: %w", err)
	}

	var racine map[string]json.RawMessage
	if err := json.Unmarshal(raw, &racine); err != nil {
		return fmt.Errorf("export hfsql: %s: %w", chemin, err)
	}
	if len(racine) != 1 {
		return fmt.Errorf("export hfsql: %s: %d tables à la racine, une seule attendue", chemin, len(racine))
	}
	for _, table := range racine {
		if err := json.Unmarshal(table, dest); err != nil {
			return fmt.Errorf("export hfsql: %s: %w", chemin, err)
		}
	}
	return nil
}

// champ nettoie une valeur texte HFSQL, stockée à largeur fixe et donc
// complétée d'espaces jusqu'au bout du champ.
func champ(s string) string { return strings.TrimSpace(s) }

// Garde-fou de compilation : Source doit satisfaire source.Source.
var _ source.Source = (*Source)(nil)
