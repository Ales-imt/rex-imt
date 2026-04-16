package cohorte

import (
	"back-rex-common/pkg/services"
	"encoding/json"
	"io"

	"net/http"

	"github.com/go-chi/render"
)

type Eleve struct {
	ID     int64  `json:"id"`
	Det    string `json:"det"`
	Nom    string `json:"nom"`
	Prenom string `json:"prenom"`
	Mail   string `json:"mail"`
	Promo  int64  `json:"promo"`
}

type DataExport struct {
	Promotions    []CreationPromotionParams `json:"promotions"`
	Eleves        []Eleve                   `json:"eleves"`
	Groupes       []CreationGroupeParams    `json:"groupes"`
	ElevesGroupes []AddEleveToGroupeParams  `json:"eleves_groupes"`
}

func ImportCohorte(w http.ResponseWriter, r *http.Request, ldapConfig services.LDAPConfig) {
	// Parse le multipart form (taille max 10 Mo ici)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// Récupère le fichier (le champ doit s'appeler "data")
	dataFile, err := getFile("data", r)
	if err != nil {
		services.InvalidRequestError(w, r, "pas de fichier data", services.NO_INFORMATION, nil)
		return
	}
	defer (*dataFile).Close()

	//  Désérialiser le fichier
	jsonBytes, err := io.ReadAll(*dataFile)
	if err != nil {
		services.InvalidRequestError(w, r, "erreur de lecture du fichier multipart", services.NO_INFORMATION, nil)
		return
	}

	var data DataExport

	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		services.InvalidRequestError(w, r, "erreur de unmarshaling JSON:", services.NO_INFORMATION, nil)
		return
	}

	// warns, err := checkLdapEleves(&data.Eleves, ldapConfig)
	// if err != nil {
	// 	services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
	// 	return
	// }

	err = toDB(r, data)
	if err != nil {
		// ne peux avoir qu'une erreur interne
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	// Réponse formatée avec items et itemCount
	w.WriteHeader(http.StatusNoContent)
}

func GetPromo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pgctx := services.GetPgCtx(ctx)
	queries := New(pgctx.Db)

	promos, err := queries.GetPromotions(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if promos == nil {
		promos = []Promotion{}
	}

	render.JSON(w, r, promos)
}

func GetGroupe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pgctx := services.GetPgCtx(ctx)
	queries := New(pgctx.Db)

	groupes, err := queries.GetGroupe(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if groupes == nil {
		groupes = []Groupe{}
	}

	render.JSON(w, r, groupes)
}
