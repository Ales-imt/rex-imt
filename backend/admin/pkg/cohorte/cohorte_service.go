package cohorte

import (
	userAdmin "back-rex-admin/pkg/user"
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	userCommon "back-rex-common/pkg/user"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func getFile(name string, r *http.Request) (*multipart.File, error) {
	// Récupère le fichier (le champ doit s'appeler "file")
	file, header, err := r.FormFile(name)
	if err != nil {
		return nil, errors.New("fichier manquant:cohortes")
	}

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".json") {
		return nil, errors.New("le fichier n'a pas l'extension .json")
	}

	return &file, nil
}

// func checkLdapEleves(eleves *[]Eleve, ldapConfig services.LDAPConfig) ([]string, error) {
// 	var warns []string

// 	// Connexion au serveur LDAP
// 	l, err := ldap.DialURL(ldapConfig.URL)
// 	if err != nil {
// 		return nil, services.NewAppInternalError("LDAP connection failed: %v", err)
// 	}

// 	defer l.Close()

// 	for _, e := range *eleves {
// 		err := isInLdap(e.Mail, l, ldapConfig.BaseDN)
// 		if err != nil {
// 			warns = append(warns, fmt.Sprintf("ldap : %v", err))
// 			continue
// 		}

// 	}

// 	return warns, nil
// }

// func isInLdap(mail string, l *ldap.Conn, baseDN string) error {

// 	filter := fmt.Sprintf("(mail=%s)", ldap.EscapeFilter(mail))

// 	searchRequest := ldap.NewSearchRequest(
// 		baseDN,
// 		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
// 		filter,
// 		[]string{"*"}, // si nil, retourne ts les attibuts.
// 		nil,
// 	)

// 	sr, err := l.Search(searchRequest)
// 	if err != nil {
// 		return fmt.Errorf("LDAP search failed: %v", err)
// 	}
// 	if len(sr.Entries) != 1 {
// 		return fmt.Errorf("utilisateur inconnu: %s", mail)
// 	}

// 	return nil

// }

func toDB(r *http.Request, data DataExport) error {
	ctx := r.Context()
	pgCtx := services.GetPgCtx(ctx)

	tx, err := pgCtx.Db.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	queries := New(tx)

	if err = promoToBd(queries, ctx, data.Promotions); err != nil {
		return err
	}

	if err = groupeToBd(queries, ctx, data.Groupes); err != nil {
		return err
	}
	var m map[int]int
	if m, err = createEtudiants(tx, queries, ctx, data.Eleves); err != nil {
		return err
	}

	if err = groupOfEtudiant(queries, ctx, m, data.ElevesGroupes); err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func groupOfEtudiant(queries *Queries, ctx context.Context, m map[int]int, elevesGroupes []AddEleveToGroupeParams) error {

	if err := queries.DeleteEleveToGroupe(ctx); err != nil {
		return err
	}

	for _, eg := range elevesGroupes {
		localkey := m[int(eg.NumEtudiant)]
		err := queries.AddEleveToGroupe(ctx, AddEleveToGroupeParams{
			NumEtudiant: int32(localkey),
			IDGroupe:    eg.IDGroupe,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func promoToBd(queries *Queries, ctx context.Context, promotions []CreationPromotionParams) error {

	for _, p := range promotions {
		if err := queries.CreationPromotion(ctx, p); err != nil {
			return err
		}
	}

	return nil
}

func groupeToBd(queries *Queries, ctx context.Context, groupes []CreationGroupeParams) error {

	for _, g := range groupes {
		if err := queries.CreationGroupe(ctx, g); err != nil {
			return err
		}
	}

	return nil
}

func createEtudiants(tx pgx.Tx, queries *Queries, ctx context.Context, eleves []Eleve) (map[int]int, error) {

	// doit regarder si l'utilisateur existe deja
	// Une erreur provoque la fin de la transaction
	queriesUerAdmin := userAdmin.New(tx)

	var cybToInternalKey map[int]int
	cybToInternalKey = make(map[int]int)

	for _, e := range eleves {

		u, err := queriesUerAdmin.GetUserByMail(ctx, e.Mail)
		var idLocal int

		switch err {
		case nil:
			// met a jour la promo de l'etudiant
			p, err := queries.GetPromotionById(ctx, e.Promo)
			if err != nil {
				return nil, err
			}

			queries.UpdateStudentPromo(ctx, UpdateStudentPromoParams{
				ID:        u.ID,
				Promotion: p.Name,
			})
			idLocal = int(u.ID)

		case pgx.ErrNoRows:

			p, err := queries.GetPromotionById(ctx, e.Promo)
			if err != nil {
				return nil, err
			}
			ident := auth.LdapIdentity{
				Name:      e.Nom,
				Surname:   e.Prenom,
				Mail:      e.Mail,
				Promotion: p.Name.String,
			}
			idLocal, err = userCommon.CreateUser(tx, &ident, ctx, "etudiant", true)
			if err != nil {
				return nil, err
			}
		default:
			return nil, err
		}

		cybToInternalKey[int(e.ID)] = idLocal
	}
	return cybToInternalKey, nil

}
