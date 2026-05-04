package note

import "context"

// NoteConnector est l'interface que tout connecteur de notes doit implémenter.
type NoteConnector interface {
	GetNotes(ctx context.Context, email string) ([]Periode, error)
}

type Matiere struct {
	Nom         string  `json:"nom"`
	Note        float64 `json:"note"`
	Coefficient float64 `json:"coefficient"`
	Date        string  `json:"date"`
	Commentaire *string `json:"commentaire,omitempty"`
}

type UE struct {
	Nom         string    `json:"nom"`
	Score       float64   `json:"score"`
	Coefficient float64   `json:"coefficient"`
	Matieres    []Matiere `json:"matieres"`
}

type Periode struct {
	Nom string  `json:"nom"`
	GPA float64 `json:"gpa"`
	UEs []UE    `json:"ues"`
}
