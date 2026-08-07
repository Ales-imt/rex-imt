package user

// Lecture de la date de sortie d'un étudiant dans Auréga (MariaDB), pour
// décider si l'horizon de conservation des pièces de présence est échu.
//
// Comportement « fail-safe » : toute incertitude — Auréga injoignable, étudiant
// inconnu, date illisible — se traduit par « date indéterminée », ce que
// account.Resolve interprète comme « on conserve ». Dans le doute, on ne
// détruit pas une identité qui pourrait encore être nécessaire pour rattacher
// une pièce de présence à une personne.

import (
	"back-rex-admin/pkg/account"
	mgen "back-rex-admin/pkg/rgpd/mariadb/gen"
	"context"
	"database/sql"
	"log"
	"time"
)

// comptesAnonymizeAfterYears doit rester aligné sur la constante homonyme de
// backend/admin/pkg/rgpd : c'est le même horizon de conservation, appliqué ici
// à la demande explicite d'un administrateur plutôt qu'au cycle automatique.
const comptesAnonymizeAfterYears = 10

// auregaSortie encapsule la connexion Auréga. Un pointeur nil, ou un champ db
// nil, signifie « source indisponible » et non « erreur » : c'est un état de
// fonctionnement normal, l'admin doit rester utilisable sans Auréga.
type auregaSortie struct {
	db *sql.DB
}

// NewAurega construit l'accès aux sorties. db peut être nil.
func NewAurega(db *sql.DB) *auregaSortie {
	return &auregaSortie{db: db}
}

// lookup retourne la fonction de résolution attendue par account.Resolve.
func (a *auregaSortie) lookup() account.SortieLookup {
	if a == nil || a.db == nil {
		log.Println("[user] Auréga indisponible : aucun compte ne sera anonymisé sur demande admin")
		return func(context.Context, string) (time.Time, bool) { return time.Time{}, false }
	}

	return func(ctx context.Context, email string) (time.Time, bool) {
		raw, err := mgen.New(a.db).GetDateFinByEmail(ctx, sql.NullString{String: email, Valid: true})
		if err != nil {
			// Inclut sql.ErrNoRows : étudiant absent d'Auréga.
			return time.Time{}, false
		}
		datefin, ok := raw.(time.Time)
		if !ok {
			return time.Time{}, false
		}
		return datefin, true
	}
}
