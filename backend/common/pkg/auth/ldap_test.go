package auth

import (
	"back-rex-common/pkg/services"
	"fmt"
	"testing"
)

func TestLdapPwd(t *testing.T) {
	cfg := services.LDAPConfig{
		URL: "ldap://localhost:3890",
		//URL:    "ldap://ldap.mines-ales.fr:389",
		BaseDN: "dc=ema,dc=fr",
	}

	pwd := "password"

	id, err := LdapAuthenticate(cfg, "Joel.vlasak@mines-ales.fr", pwd)

	if err != nil {
		t.Errorf("1-LdapAuthenticate failed for email: %v", err)
	}
	fmt.Println(id)

	_, err = LdapAuthenticate(cfg, "vlasak", pwd)
	if err != nil {
		t.Errorf("2-LdapAuthenticate failed for email: %v", err)
	}
}
