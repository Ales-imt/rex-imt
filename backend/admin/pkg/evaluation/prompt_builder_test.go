package evaluation

import (
	"strings"
	"testing"
)

func ptr(f float64) *float64 { return &f }

func TestBuildPrompt_Nominal(t *testing.T) {
	data := PromptData{
		Eval: PromptEvalData{
			MatiereName:   "Machine Learning",
			PromotionName: "IMT 2025",
			PeriodeName:   "S5",
			NbRepondants:  7,
			ScorePeda:     ptr(4.2),
			ScoreClarte:   ptr(3.9),
			ScoreContenu:  ptr(4.5),
			ScoreSupports: ptr(4.0),
			ScoreAmbiance: ptr(4.1),
			NpsMoyen:      ptr(8.5),
		},
		Chips: []PromptChip{
			{Libelle: "Cours bien structurés", Polarite: "POSITIF", Nb: 5},
			{Libelle: "Manque d'exemples pratiques", Polarite: "NEGATIF", Nb: 3},
		},
		Verbatims: []PromptVerbatim{
			{Dimension: "SUPPORTS", Texte: "Les slides sont très clairs."},
			{Dimension: "AMBIANCE", Texte: "Bonne ambiance en cours."},
		},
	}

	prompt := BuildPrompt(data)

	checks := []string{
		"Machine Learning",
		"IMT 2025",
		"S5",
		"7",
		"4.20/5",
		"8.5/10",
		"Cours bien structurés (5 mentions)",
		"Manque d'exemples pratiques (3 mentions)",
		"[SUPPORTS] « Les slides sont très clairs. »",
		"Bilan général",
		"Points forts",
		"Axes d'amélioration",
		"Recommandations",
	}
	for _, want := range checks {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt ne contient pas %q", want)
		}
	}
}

func TestBuildPrompt_NoVerbatims(t *testing.T) {
	data := PromptData{
		Eval: PromptEvalData{
			MatiereName:   "Algo",
			PromotionName: "IMT 2024",
			PeriodeName:   "S3",
			NbRepondants:  5,
		},
	}

	prompt := BuildPrompt(data)

	if !strings.Contains(prompt, "Aucun signal positif notable") {
		t.Error("prompt doit indiquer l'absence de signaux positifs")
	}
	if !strings.Contains(prompt, "Aucun signal négatif notable") {
		t.Error("prompt doit indiquer l'absence de signaux négatifs")
	}
	if strings.Contains(prompt, "Verbatims") {
		t.Error("prompt ne doit pas afficher la section verbatims si elle est vide")
	}
	if !strings.Contains(prompt, "N/A") {
		t.Error("prompt doit afficher N/A pour les scores absents")
	}
}
