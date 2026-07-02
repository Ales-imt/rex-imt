package annee

import (
	"back-rex-common/pkg/services"
	"context"
	"log"
	"net/http"
)

var AnneeContextKey = &services.ContextKey{Name: "Annee"}

func getAnneeFromCtx(r *http.Request) *Annee {
	annee, ok := r.Context().Value(AnneeContextKey).(*Annee)
	if ok {
		return annee
	}
	log.Fatal("annee inconnue")
	return nil
}

func setAnneeFromCtx(r *http.Request, annee *Annee) context.Context {
	return context.WithValue(r.Context(), AnneeContextKey, annee)
}
