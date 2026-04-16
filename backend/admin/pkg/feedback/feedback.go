package feedback

import (
	"back-rex-admin/pkg/ia"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

type Classification struct {
	FeedbackID int    `json:"feedback_id"`
	Categorie  string `json:"categorie"`
	SousCateg  string `json:"sous_categorie"`
	Sentiment  string `json:"sentiment"` // positif | negatif | neutre | mitige
	Urgence    int    `json:"urgence"`   // 1 (faible) à 5 (critique)
	Resume     string `json:"resume"`
}

func GetAllFeedback(w http.ResponseWriter, r *http.Request) {
	pgctx := services.GetPgCtx(r.Context())
	query := New(pgctx.Db)

	feedbacks, err := query.ListFeedbacks(context.Background())
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if feedbacks == nil {
		feedbacks = []ListFeedbacksRow{}
	}
	// Réponse formatée avec items et itemCount
	render.JSON(w, r, feedbacks)
}

func CallbackFeedBack(notif FeedbackNotification, connector ia.IAConnector) {
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
		fmt.Printf("parse JSON classification: %w\nRéponse brute: %s", err, cleaned)
	}

	fmt.Println(cleaned)

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
