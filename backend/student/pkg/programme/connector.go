package programme

import "context"

// ProgrammeConnector est l'interface que tout connecteur de programme doit implémenter.
type ProgrammeConnector interface {
	// GetProgramme retourne les cours pour la plage start–end (format YYYYMMDD).
	// Pour un utilisateur normal, le planning est filtré sur son email ; pour un
	// gestionnaire (gestionnaire=true), toutes les séances de la période sont
	// retournées, sans filtrage par utilisateur.
	GetProgramme(ctx context.Context, email, start, end string, gestionnaire bool) ([]Cours, error)
}

// Cours est une séance planifiée. Cocle identifie la matière dans webdfd.
type Cours struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	HD    string `json:"hd"`    // HH:MM
	HF    string `json:"hf"`    // HH:MM
	Cocle string `json:"cocle"` // identifiant matière webdfd
	Cours string `json:"cours"` // nom affiché
	Salle string `json:"salle"`
	Prof  string `json:"prof"`
	Promo string `json:"promo"`
}
