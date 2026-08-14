package watcher

import (
	"bytes"
	"os"
	"testing"
)

// L'analyse embarquee est la seule ressource du binaire : si elle est vide ou
// remplacee par un autre fichier, l'export echoue silencieusement cote WinDev.
func TestEmptempEmbarquee(t *testing.T) {
	if len(emptemp) == 0 {
		t.Fatal("analyse embarquée vide")
	}
	if !bytes.Contains(emptemp[:64], []byte("emptemp.WD5")) {
		t.Fatal("l'analyse embarquée ne ressemble pas à un fichier WDD")
	}

	surDisque, err := os.ReadFile(emptempName)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(emptemp, surDisque) {
		t.Fatalf("l'analyse embarquée diffère de %s", emptempName)
	}
}
