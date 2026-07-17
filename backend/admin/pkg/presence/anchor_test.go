package presence_test

import (
	"back-rex-common/pkg/services"
	"testing"
)

// Les tests du chaînage (VerifyChain, GetLastLedgerEntry, ComputeHash) sont
// dans back-rex-common/pkg/ledger, seule implémentation du registre.

func TestDefaultTSAURL(t *testing.T) {
	if services.DefaultTSAURL == "" {
		t.Fatal("DefaultTSAURL must not be empty")
	}
}
