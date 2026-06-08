package auth

import (
	"back-rex-common/pkg/services"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Vérifie l'authentification d'un utilisateur LDAP
func LdapAuthenticate(cfg services.LDAPConfig, identifiant string, password string) (*LdapIdentity, error) {

	if identifiant == "clement.trens@etu.mines-ales.fr" && password == "t" {
		identity := LdapIdentity{Name: "T2", Surname: "T2 user",
			Mail: "clement.trens@etu.mines-ales.fr", Promotion: "Infres"}
		// Simule une authentification réussie pour les tests
		return &identity, nil // Simule une authentification réussie pour les tests
	}

	// Connexion au serveur LDAP
	l, err := ldap.DialURL(cfg.URL)
	if err != nil {
		log.Printf("LDAP connection failed: %v", err)
		return nil, services.NewAppInternalError("Erreur d'authentification")
	}
	defer l.Close()

	identifiant = strings.ToLower(identifiant)

	var filter string
	if services.IsValidEmail(identifiant) {
		filter = fmt.Sprintf("(mail=%s)", ldap.EscapeFilter(identifiant))
	} else {
		filter = fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(identifiant))
	}

	searchRequest := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		filter,
		[]string{"*"}, // si nil, retourne ts les attibuts.
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Printf("LDAP search failed for %q: %v", identifiant, err)
		return nil, services.NewAppInternalError("Erreur d'authentification")
	}
	if len(sr.Entries) != 1 {
		return nil, services.NewAppValidationError("Identifiant ou mot de passe incorrect", "identifiant")
	}

	userDN := sr.Entries[0].DN

	// Tente de se binder avec le DN et le mot de passe
	err = l.Bind(userDN, password)
	if err != nil {
		return nil, services.NewAppValidationError("Identifiant ou mot de passe incorrect", "identifiant")
	}

	return GetLdapIdentity(sr.Entries[0]), nil // Authentification réussie
}

func GetLdapIdentity(entry *ldap.Entry) *LdapIdentity {
	identity := LdapIdentity{}

	identity.Name = entry.GetAttributeValue("sn")
	identity.Surname = entry.GetAttributeValue("givenName")
	identity.Mail = strings.ToLower(entry.GetAttributeValue("mail"))
	identity.Promotion = entry.GetAttributeValue("ou")
	return &identity
}

type LdapIdentity struct {
	Name      string
	Surname   string
	Mail      string
	Promotion string
}

type LdapPostHandler func(r *http.Request, ldapIdentity *LdapIdentity) (*jwt.MapClaims, *string, error)
