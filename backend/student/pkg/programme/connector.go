package programme

import (
	"context"
	"time"
)

// Demandeur décrit l'utilisateur qui consulte le planning.
//
// Le connecteur BD travaille sur un user_id et un rôle (les séances sont
// rattachées à public."user"), là où webdfd et aurega résolvent leur
// interlocuteur par son email. Les trois informations voyagent donc ensemble
// plutôt que d'imposer à chaque connecteur une résolution qui ne le concerne
// pas.
type Demandeur struct {
	UserID int32
	Email  string
	Roles  []string
}

// ProgrammeConnector est l'interface que tout connecteur de programme doit implémenter.
type ProgrammeConnector interface {
	// GetProgramme retourne les cours dont le début tombe dans [debut, fin).
	// La borne haute est EXCLUSIVE : les connecteurs qui parlent à un système
	// amont attendant une date de fin inclusive doivent la reculer d'un jour.
	//
	// Le filtrage dépend du rôle du demandeur : un gestionnaire voit toutes les
	// séances de la plage, un prof les siennes, un élève celles de ses groupes.
	GetProgramme(ctx context.Context, d Demandeur, debut, fin time.Time) ([]Cours, error)
}

// Cours est une séance planifiée. Cocle identifie la matière dans webdfd.
type Cours struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	HD    string `json:"hd"`    // HH:MM
	HF    string `json:"hf"`    // HH:MM
	Cocle string `json:"cocle"` // identifiant matière webdfd
	// MatiereID est l'identifiant interne de la matière (public.matiere), que
	// seul le connecteur BD sait renseigner : `cocle` est une clé webdfd, sans
	// signification pour lui. Ajout additif, omis quand il vaut 0 — le front
	// mobile existant continue de ne lire que les champs qu'il connaît.
	MatiereID int64  `json:"matiere_id,omitempty"`
	Cours     string `json:"cours"` // nom affiché
	Salle     string `json:"salle"`
	Prof      string `json:"prof"`
	Promo     string `json:"promo"`
}
