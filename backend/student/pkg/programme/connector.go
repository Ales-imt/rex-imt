package programme

import "context"

// ProgrammeConnector est l'interface que tout connecteur de programme doit implémenter.
type ProgrammeConnector interface {
	// GetProgramme retourne les cours d'un étudiant pour la plage start–end (format YYYYMMDD).
	GetProgramme(ctx context.Context, email, start, end string) ([]Cours, error)
}

// Cours est l'évènement renvoyé au front.
type Cours struct {
	Date  string `json:"date"` // YYYY-MM-DD
	HD    string `json:"hd"`   // HH:MM
	HF    string `json:"hf"`   // HH:MM
	Cours string `json:"cours"`
	Salle string `json:"salle"`
	Prof  string `json:"prof"`
	Promo string `json:"promo"`
}
