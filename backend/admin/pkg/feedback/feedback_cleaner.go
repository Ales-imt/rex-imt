package feedback

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// hasConsecutiveRepeat retourne true si un même rune apparaît
// plus de `max` fois consécutivement dans s.
// Remplace le regex `(.)\1{N,}` incompatible avec RE2 (moteur Go).
func hasConsecutiveRepeat(s string, max int) bool {
	runes := []rune(s)
	count := 1
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			count++
			if count > max {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

// Feedback représente un message brut issu du fichier JSON
type Feedback struct {
	ID        int    `json:"ID"`
	Content   string `json:"Content"`
	CreatedAt string `json:"CreatedAt"`
	Promotion string `json:"Promotion"`
}

// CleanedFeedback représente un message après filtrage et nettoyage
type CleanedFeedback struct {
	ID        int    `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	Promotion string `json:"promotion"`
}

// FilterResult contient les statistiques du pipeline de nettoyage
type FilterResult struct {
	Total    int
	Kept     int
	Rejected int
	Reasons  map[string]int
}

const (
	minLength     = 15 // Nombre minimum de caractères
	minWordCount  = 3  // Nombre minimum de mots
	maxRepeatChar = 5  // Répétition max d'un même caractère (ex: "aaaaaa")
)

// Mots et patterns qui signalent un message sans valeur
var (
	spamPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(test|essai|essaie|feedback|bonjour|oui|non|ok|coucou|pouet|bruh|appli|camarche)$`),
		regexp.MustCompile(`(?i)^[a-z]$`),                     // Lettre isolée
		regexp.MustCompile(`(?i)^[:;()\-!?.]+$`),              // Smiley pur ":)"
		regexp.MustCompile(`(?i)^(test.{0,10}|essai.{0,5})$`), // "test sans wifi", "essaie"
	}

	// Nettoyage du texte
	multiSpaces   = regexp.MustCompile(`\s{2,}`)
	leadingSpaces = regexp.MustCompile(`(?m)^\s+`)
)

// isSpam détecte les messages sans valeur sémantique
func isSpam(content string) (bool, string) {
	trimmed := strings.TrimSpace(content)
	lower := strings.ToLower(trimmed)

	// Trop court
	if utf8.RuneCountInString(trimmed) < minLength {
		return true, "trop_court"
	}

	// Trop peu de mots
	words := strings.Fields(trimmed)
	if len(words) < minWordCount {
		return true, "trop_peu_de_mots"
	}

	// Proportion de lettres trop faible (messages de ponctuation, chiffres seuls...)
	letterCount := 0
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			letterCount++
		}
	}
	ratio := float64(letterCount) / float64(utf8.RuneCountInString(trimmed))
	if ratio < 0.5 {
		return true, "peu_de_lettres"
	}

	// Patterns spam explicites
	for _, re := range spamPatterns {
		if re.MatchString(lower) {
			return true, "pattern_spam"
		}
	}

	// Répétition excessive d'un caractère (ex: "amamamamama")
	// RE2 (Go) ne supporte pas les backreferences — on utilise une fonction native
	if hasConsecutiveRepeat(lower, maxRepeatChar) {
		return true, "repetition_caractere"
	}

	return false, ""
}

// cleanContent normalise le texte d'un feedback
func cleanContent(content string) string {
	// Suppression des espaces multiples et en début de ligne
	content = multiSpaces.ReplaceAllString(content, " ")
	content = leadingSpaces.ReplaceAllString(content, "")

	// Suppression des retours à la ligne multiples (garde les simples)
	lines := strings.Split(content, "\n")
	var cleaned []string
	prev := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" && prev == "" {
			continue // Supprime les lignes vides consécutives
		}
		cleaned = append(cleaned, line)
		prev = line
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// CleanSingleFeedback filtre et nettoie un seul feedback.
// Retourne le contenu nettoyé et true si le feedback est valide, false s'il est spam.
func CleanSingleFeedback(content string) (string, bool) {
	spam, _ := isSpam(content)
	if spam {
		return "", false
	}
	return cleanContent(content), true
}
