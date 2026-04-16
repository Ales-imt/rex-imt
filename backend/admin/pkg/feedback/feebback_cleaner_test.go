package feedback

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestFedbackCleaner(t *testing.T) {
	FeedbackCleaner()
}

func FeedbackCleaner() {

	inputPath := "feedback.json"
	outputPath := "feedback_cleaned.json"

	fmt.Printf("Traitement de %s...\n", inputPath)

	feedbacks, stats, err := ProcessFeedbacks(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}

	// Écriture du fichier nettoyé
	out, err := json.MarshalIndent(feedbacks, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur sérialisation: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputPath, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur écriture: %v\n", err)
		os.Exit(1)
	}

	// Rapport de nettoyage
	fmt.Println("\n--- Rapport de nettoyage ---")
	fmt.Printf("Total messages     : %d\n", stats.Total)
	fmt.Printf("Conservés          : %d (%.1f%%)\n", stats.Kept, float64(stats.Kept)/float64(stats.Total)*100)
	fmt.Printf("Rejetés            : %d (%.1f%%)\n", stats.Rejected, float64(stats.Rejected)/float64(stats.Total)*100)
	fmt.Println("\nRaisons de rejet :")
	for reason, count := range stats.Reasons {
		fmt.Printf("  %-25s : %d\n", reason, count)
	}
	fmt.Printf("\nFichier écrit : %s\n", outputPath)
}

// processFeedbacks charge, filtre et nettoie les feedbacks
func ProcessFeedbacks(inputPath string) ([]CleanedFeedback, FilterResult, error) {
	result := FilterResult{Reasons: make(map[string]int)}

	// Lecture du fichier JSON
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, result, fmt.Errorf("lecture fichier: %w", err)
	}

	var raw []Feedback
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, result, fmt.Errorf("parsing JSON: %w", err)
	}

	result.Total = len(raw)

	var kept []CleanedFeedback
	for _, fb := range raw {
		spam, reason := isSpam(fb.Content)
		if spam {
			result.Rejected++
			result.Reasons[reason]++
			continue
		}

		kept = append(kept, CleanedFeedback{
			ID:        fb.ID,
			Content:   cleanContent(fb.Content),
			CreatedAt: fb.CreatedAt,
			Promotion: fb.Promotion,
		})
	}

	result.Kept = len(kept)
	return kept, result, nil
}
