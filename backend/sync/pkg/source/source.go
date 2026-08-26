package source

// Noms de source reconnus par la configuration (migration.source).
const (
	// NomWebdfd interroge cybema en direct, en HTTP (cgiempt.exe/cgihtml.exe).
	NomWebdfd = "webdfd"
	// NomHFSQL lit les exports JSON produits par le service export-webdfd
	// à partir des sauvegardes HFSQL de cybema.
	NomHFSQL = "hfsql"
)

// Map est la valeur écrite dans la colonne `source` des tables
// migration.*_map, quelle que soit la source configurée.
//
// Elle désigne le RÉFÉRENTIEL amont — cybema — et non le transport : webdfd et
// hfsql servent les mêmes identifiants, qui sont les clés primaires HFSQL
// (P0CLEUNIK = P0, EVCLEUNIK = EV, PLCLEUNIK = PL…). Les distinguer créerait un
// doublon de chaque utilisateur, promotion et séance au premier basculement de
// source, et obligerait à paramétrer les requêtes qui filtrent dessus
// (query.sql : ListPromotionWebdfdIDs, ListGroupeWebdfdIDs,
// ListSeancesWebdfdNonVues).
const Map = "webdfd"

// Creneau regroupe les données utiles d'un créneau du planning amont,
// quelle que soit la source qui l'a produit.
//
// Les tags JSON ne servent PAS à lire les sources — hfsql décode l'export dans
// ses propres types puis convertit — mais à écrire le dump de diagnostic
// (pkg/migration/dump.go), qu'ils rendent lisible et interrogeable à `jq`.
type Creneau struct {
	Plcle  string `json:"plcle"`  // identifiant stable du créneau (PL / PLCLEUNIK)
	P0cle  string `json:"p0cle"`  // promo (P0CLE / P0CLEUNIK)
	Cocle  string `json:"cocle"`  // matière (COCLE / COCLEUNIK)
	Grcle  string `json:"grcle"`  // groupe (GRCLE / GRCLEUNIK)
	Prcle  string `json:"prcle"`  // identifiant stable du prof (PRCLE / PRCLEUNIK)
	Cours  string `json:"cours"`  // nom d'affichage de la matière (tronqué)
	Groupe string `json:"groupe"` // nom d'affichage du groupe
	Date   string `json:"date"`   // AAAAMMJJ
	HD     string `json:"hd"`     // HHMM heure de début
	HF     string `json:"hf"`     // HHMM heure de fin
	Salle  string `json:"salle"`  // salle
	Prof   string `json:"prof"`   // nom d'affichage du professeur
	Note   string `json:"note"`   // note libre du créneau (LANOTE / note), vide si absente
}

// Promo est une promotion du référentiel amont.
type Promo struct {
	ExternalID string // P0CLEUNIK
	Nom        string
}

// Personne est un professeur ou un élève du référentiel amont.
type Personne struct {
	ExternalID string // PRCLEUNIK pour un prof, EVCLEUNIK pour un élève
	Nom        string
	Prenom     string
	Email      string
}

// Source fournit le référentiel scolaire amont à synchroniser. Les deux
// implémentations — webdfd (HTTP) et hfsql (exports JSON) — décrivent la même
// base cybema et rendent donc les mêmes identifiants.
//
// Toutes les chaînes rendues sont déjà en UTF-8 et débarrassées de leurs
// espaces de remplissage : les particularités de format (Windows-1252 pour
// webdfd, largeur fixe pour HFSQL) ne remontent pas jusqu'aux étapes de
// synchronisation.
type Source interface {
	// Promos rend les promotions connues.
	Promos() ([]Promo, error)
	// Profs rend les professeurs connus.
	Profs() ([]Personne, error)
	// Eleves rend les élèves connus.
	Eleves() ([]Personne, error)
	// CoursNoms rend les noms complets des matières, indexés par COCLEUNIK.
	CoursNoms() (map[string]string, error)
	// Planning rend les créneaux d'une promotion sur une plage de dates
	// exprimée en AAAAMMJJ, bornes comprises.
	Planning(p0cle, debut, fin string) ([]Creneau, error)
	// MembresGroupe rend les EVCLEUNIK des élèves d'un groupe, et le libellé du
	// groupe tel que l'amont le connaît — vide si la source ne le fournit pas.
	MembresGroupe(grcle string) (evs []string, libelle string, err error)
}
