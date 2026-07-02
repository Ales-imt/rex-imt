package migration

import "testing"

func TestSemesterFromName(t *testing.T) {
	cases := []struct {
		nom      string
		wantName string
		wantOK   bool
	}{
		{"9.2.1 PROJET INTÉGRATEUR", "S9", true},
		{"1.1 ANGLAIS", "S1", true},
		{"RATTRAPAGE", "INCONNU", false},
		{"0.1 COURS", "INCONNU", false},
		{"", "INCONNU", false},
		{"abc.def NOM", "INCONNU", false},
	}
	for _, tc := range cases {
		got, ok := semesterFromName(tc.nom)
		if ok != tc.wantOK || got != tc.wantName {
			t.Errorf("semesterFromName(%q) = (%q, %v), want (%q, %v)",
				tc.nom, got, ok, tc.wantName, tc.wantOK)
		}
	}
}
