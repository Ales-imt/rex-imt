package presencetoken

import (
	"strings"
	"testing"
)

func TestIssueVerify_RoundTrip(t *testing.T) {
	SetSecret("secret-de-test")
	token := Issue(42)

	id, err := Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if id != 42 {
		t.Fatalf("seance_id = %d, want 42", id)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	SetSecret("secret-A")
	token := Issue(42)

	SetSecret("secret-B")
	if _, err := Verify(token); err == nil || err.Error() != "signature invalide" {
		t.Fatalf("expected 'signature invalide', got %v", err)
	}
}

func TestVerify_Garbage(t *testing.T) {
	SetSecret("secret-de-test")
	for _, tok := range []string{"", "abc", "a.b.c", "!!!.???"} {
		if _, err := Verify(tok); err == nil {
			t.Errorf("Verify(%q) should fail", tok)
		}
	}
}

func TestGenerateCode_Format(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		code := GenerateCode()
		if len(code) != 6 {
			t.Fatalf("code %q: length %d, want 6", code, len(code))
		}
		if strings.ContainsAny(code, "ILO01") {
			t.Fatalf("code %q contains ambiguous characters", code)
		}
		seen[code] = true
	}
	if len(seen) < 90 {
		t.Fatalf("only %d distinct codes out of 100", len(seen))
	}
}
