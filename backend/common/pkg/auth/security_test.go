package auth

import "testing"

// TestContainsRole_ProfAllowedOnStudentAccess vérifie le mécanisme exact
// utilisé par backend/student/cmd/main.go pour ouvrir les routes student aux
// profs (chantier 4) : studentAccess := []string{RoleEleve, RoleProf}.
func TestContainsRole_ProfAllowedOnStudentAccess(t *testing.T) {
	studentAccess := []string{RoleEleve, RoleProf}

	if !containsRole(RoleProf, studentAccess) {
		t.Fatal("un PROF (interne ou externe) doit accéder aux routes ouvertes à ELEVE et PROF")
	}
	if !containsRole(RoleEleve, studentAccess) {
		t.Fatal("un ELEVE doit continuer à accéder aux routes student")
	}
	if containsRole(RoleAdmin, studentAccess) {
		t.Fatal("un ADMIN seul ne doit pas obtenir un accès student implicite")
	}
}

// TestContainsRole_MultiRoleUser : le claim JWT "roles" est une liste
// séparée par des virgules ; l'accès doit être accordé dès qu'un des rôles
// de l'utilisateur est autorisé.
func TestContainsRole_MultiRoleUser(t *testing.T) {
	if !containsRole("ELEVE,PROF", []string{RoleProf}) {
		t.Fatal("attendu : accès autorisé via le rôle PROF parmi plusieurs rôles")
	}
	if containsRole("GESTIONNAIRE,ADMIN", []string{RoleEleve, RoleProf}) {
		t.Fatal("attendu : refus quand aucun rôle de l'utilisateur n'est autorisé")
	}
}
