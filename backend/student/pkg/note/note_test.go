package note_test

import (
	"back-rex-common/pkg/services"
	"back-rex-eleve/pkg/note"
	mariadbnote "back-rex-eleve/pkg/note/mariadb"
	"context"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

type testConfig struct {
	TestUserEmail string `yaml:"test_user_email"`
}

func loadTestConfig(t *testing.T) testConfig {
	t.Helper()
	data, err := os.ReadFile("testconfig.yaml")
	if err != nil {
		t.Skip("testconfig.yaml introuvable, test ignoré")
	}
	var cfg testConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("lecture testconfig.yaml: %v", err)
	}
	return cfg
}

func TestGetNotes_ClementTrens(t *testing.T) {
	cfg := loadTestConfig(t)

	db, err := services.NewMariaDBConnection(services.MariaDBConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "root",
		Database: "cyber_notes_v2",
	})
	if err != nil {
		t.Fatalf("connexion MariaDB échouée: %v", err)
	}
	defer db.Close()

	connector := &mariadbnote.Connector{DB: db}
	periodes, err := connector.GetNotes(context.Background(), cfg.TestUserEmail)
	if err != nil {
		t.Fatalf("GetNotes échoué: %v", err)
	}

	for _, p := range periodes {
		t.Logf("Période: %s | GPA: %.2f | UEs: %d", p.Nom, p.GPA, len(p.UEs))
		for _, ue := range p.UEs {
			t.Logf("  UE: %s (score=%.0f coef=%.0f) | matières: %d", ue.Nom, ue.Score, ue.Coefficient, len(ue.Matieres))
		}
	}

	if len(periodes) == 0 {
		t.Error("aucune période retournée")
	}

	// pour la compilation, on vérifie que le Connector implémente bien l'interface NoteConnector
	var _ note.NoteConnector = connector
}
