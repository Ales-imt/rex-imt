package feedback

import (
	"back-rex-admin/pkg/ia"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jackc/pgx/v5"
)

func CallbackFeedBack(notif FeedbackNotification, connector ia.IAConnector, conn *pgx.Conn) {
	start := time.Now()
	defer func() {
		log.Printf("[IA] feedback #%d traité en %s", notif.ID, time.Since(start))
	}()

	cleanedFeedback, valid := CleanSingleFeedback(notif.Content)
	if !valid {
		log.Printf("[IA] feedback #%d rejeté (spam/invalide)", notif.ID)
		return
	}

	prompt := fmt.Sprintf("%s\n\nFeedback à analyser :\n%s", systemPrompt, cleanedFeedback)
	response, err := connector.Analyze(context.Background(), prompt)
	if err != nil {
		log.Printf("[IA] feedback #%d erreur analyse: %v", notif.ID, err)
		return
	}

	// Nettoyage au cas où Mistral ajoute du markdown malgré le format:json
	cleaned := strings.TrimSpace(response)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var result Classification
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		log.Printf("[IA] feedback #%d erreur parsing JSON: %v\nRéponse brute: %s", notif.ID, err, cleaned)
		return
	}

	toText := func(s *string) pgtype.Text {
		if s == nil {
			return pgtype.Text{Valid: false}
		}
		return pgtype.Text{String: *s, Valid: true}
	}

	if err := New(conn).InsertClassification(context.Background(), InsertClassificationParams{
		FeedbackID:    notif.ID,
		Categorie:     result.Categorie,
		SousCategorie: result.SousCateg,
		Sentiment:     result.Sentiment,
		Urgence:       int32(result.Urgence),
		Resume:        result.Resume,
		Promotion:     toText(notif.Promotion),
		Groupe:        toText(notif.Groupe),
	}); err != nil {
		log.Printf("[IA] feedback #%d erreur persistance: %v", notif.ID, err)
	}
}

// Catégories disponibles (issues de notre taxonomie)
const systemPrompt = `Tu es un assistant qui analyse des feedbacks d'étudiants d'une école d'ingénieurs française.
 
Réponds UNIQUEMENT avec un objet JSON valide, sans markdown, sans explication.
 
Catégories disponibles :
- pedagogie / contenu_cours
- pedagogie / methodes_format
- pedagogie / evaluation_partiels
- pedagogie / motivation_bienetre
- campus / batiments_equipements
- campus / outils_numeriques
- campus / transport_mobilite
- campus / planning_organisation
- signalement / integrite_academique
- signalement / communication_ecole
- signalement / app_trex
- autre / feedback_positif
- autre / ecologie_sens
- autre / spam

Format de réponse attendu :
{
  "categorie": "pedagogie",
  "sous_categorie": "contenu_cours",
  "sentiment": "negatif",
  "urgence": 3,
  "resume": "résumé en 10 mots max"
}
 
Règle urgence :
1 = informatif, 2 = à noter, 3 = à surveiller, 4 = action requise, 5 = alerte immédiate`
