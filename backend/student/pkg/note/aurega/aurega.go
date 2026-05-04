package aurega

import (
	"back-rex-eleve/pkg/note"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Connector récupère les notes depuis l'API Aurega.
type Connector struct {
	BaseURL string
	APIKey  string
}

type auregaRequest struct {
	Email string `json:"email"`
}

type auregaMatiere struct {
	Nom         string  `json:"nom"`
	Note        float64 `json:"note"`
	Coefficient float64 `json:"coefficient"`
	Date        string  `json:"date"`
	Commentaire *string `json:"commentaire,omitempty"`
}

type auregaUE struct {
	Nom         string          `json:"nom"`
	Score       float64         `json:"score"`
	Coefficient float64         `json:"coefficient"`
	Matieres    []auregaMatiere `json:"matieres"`
}

type auregaPeriode struct {
	Nom string     `json:"nom"`
	GPA float64    `json:"gpa"`
	UEs []auregaUE `json:"ues"`
}

func (c *Connector) GetNotes(ctx context.Context, email string) ([]note.Periode, error) {
	body, err := json.Marshal(auregaRequest{Email: email})
	if err != nil {
		return nil, fmt.Errorf("aurega: sérialisation requête: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/notes", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("aurega: création requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aurega: appel HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aurega: statut inattendu %d", resp.StatusCode)
	}

	var raw []auregaPeriode
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("aurega: décodage réponse: %w", err)
	}

	result := make([]note.Periode, len(raw))
	for i, p := range raw {
		ues := make([]note.UE, len(p.UEs))
		for j, u := range p.UEs {
			matieres := make([]note.Matiere, len(u.Matieres))
			for k, m := range u.Matieres {
				matieres[k] = note.Matiere{
					Nom:         m.Nom,
					Note:        m.Note,
					Coefficient: m.Coefficient,
					Date:        m.Date,
					Commentaire: m.Commentaire,
				}
			}
			ues[j] = note.UE{Nom: u.Nom, Score: u.Score, Coefficient: u.Coefficient, Matieres: matieres}
		}
		result[i] = note.Periode{Nom: p.Nom, GPA: p.GPA, UEs: ues}
	}
	return result, nil
}
