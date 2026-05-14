package evaluation

import (
	"fmt"
	"strings"
)

type PromptEvalData struct {
	MatiereName   string
	PromotionName string
	PeriodeName   string
	NbRepondants  int
	ScorePeda     *float64
	ScoreClarte   *float64
	ScoreContenu  *float64
	ScoreSupports *float64
	ScoreAmbiance *float64
	NpsMoyen      *float64
}

type PromptChip struct {
	Libelle  string
	Polarite string
	Nb       int
}

type PromptVerbatim struct {
	Dimension string
	Texte     string
}

type PromptData struct {
	Eval      PromptEvalData
	Chips     []PromptChip
	Verbatims []PromptVerbatim
}

func fmtScore(f *float64) string {
	if f == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.2f/5", *f)
}

func fmtNps(f *float64) string {
	if f == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.1f/10", *f)
}

func BuildPrompt(data PromptData) string {
	var sb strings.Builder

	fmt.Fprintf(&sb,
		"Tu es un assistant pédagogique expert. Génère une synthèse structurée en français (300 à 400 mots) des évaluations de la matière \"%s\" (%s, %s).\n\n",
		data.Eval.MatiereName, data.Eval.PromotionName, data.Eval.PeriodeName,
	)

	fmt.Fprintf(&sb,
		"Données clés :\n"+
			"- Nombre de répondants : %d\n"+
			"- Score pédagogique global : %s\n"+
			"- Score clarté : %s\n"+
			"- Score contenu : %s\n"+
			"- Score supports : %s\n"+
			"- Score ambiance : %s\n"+
			"- NPS moyen : %s\n\n",
		data.Eval.NbRepondants,
		fmtScore(data.Eval.ScorePeda),
		fmtScore(data.Eval.ScoreClarte),
		fmtScore(data.Eval.ScoreContenu),
		fmtScore(data.Eval.ScoreSupports),
		fmtScore(data.Eval.ScoreAmbiance),
		fmtNps(data.Eval.NpsMoyen),
	)

	var pos, neg []PromptChip
	for _, c := range data.Chips {
		if c.Polarite == "POSITIF" {
			pos = append(pos, c)
		} else {
			neg = append(neg, c)
		}
	}

	sb.WriteString("Points forts (signaux positifs) :\n")
	if len(pos) == 0 {
		sb.WriteString("- Aucun signal positif notable\n")
	} else {
		for _, c := range pos {
			fmt.Fprintf(&sb, "- %s (%d mentions)\n", c.Libelle, c.Nb)
		}
	}

	sb.WriteString("\nPoints d'amélioration (signaux négatifs) :\n")
	if len(neg) == 0 {
		sb.WriteString("- Aucun signal négatif notable\n")
	} else {
		for _, c := range neg {
			fmt.Fprintf(&sb, "- %s (%d mentions)\n", c.Libelle, c.Nb)
		}
	}

	if len(data.Verbatims) > 0 {
		sb.WriteString("\nVerbatims des étudiants :\n")
		for i, v := range data.Verbatims {
			if i >= 15 {
				break
			}
			fmt.Fprintf(&sb, "- [%s] « %s »\n", v.Dimension, v.Texte)
		}
	}

	sb.WriteString(
		"\nRédige une synthèse structurée avec les sections suivantes :\n" +
			"1. Bilan général\n" +
			"2. Points forts\n" +
			"3. Axes d'amélioration\n" +
			"4. Recommandations\n\n" +
			"Réponds directement avec la synthèse, sans introduction ni conclusion méta.",
	)

	return sb.String()
}
