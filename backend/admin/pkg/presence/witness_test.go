package presence

// Tests du témoin externe. Aucune TSA ni aucun serveur SMTP réel n'est
// contacté : l'envoi passe par un mini-serveur SMTP local monté sur un port
// éphémère (voir smtptest_test.go).
//
// Ce fichier est dans le package presence (et non presence_test) pour tester
// buildWitnessMessage, non exporté.

import (
	"back-rex-common/pkg/services"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func testWitness() Witness {
	return Witness{
		AnchorID:   12,
		Seq:        42,
		Hash:       strings.Repeat("ab", 32), // 64 hex chars
		AnchoredAt: time.Date(2026, 7, 9, 14, 30, 0, 0, time.UTC),
		TSAURL:     "https://freetsa.org/tsr",
		Token:      []byte{0x30, 0x82, 0x01, 0x00}, // pseudo-DER
		TSACertPEM: []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n"),
	}
}

// ── Construction du message ──────────────────────────────────────────────────

func TestBuildWitnessMessage_Content(t *testing.T) {
	w := testWitness()
	msg := buildWitnessMessage("noreply@exemple.fr", "archive@tiers.org", w)

	if len(msg.To) != 1 || msg.To[0] != "archive@tiers.org" {
		t.Fatalf("destinataire inattendu: %v", msg.To)
	}
	for _, want := range []string{"42", w.Hash, "2026-07-09T14:30:00Z", "https://freetsa.org/tsr", "preuve d'archivage"} {
		if !strings.Contains(msg.Body, want) {
			t.Errorf("corps sans %q:\n%s", want, msg.Body)
		}
	}
	if !strings.Contains(msg.Subject, "42") {
		t.Errorf("sujet sans le seq: %q", msg.Subject)
	}
}

func TestBuildWitnessMessage_Attachments(t *testing.T) {
	w := testWitness()
	msg := buildWitnessMessage("noreply@exemple.fr", "archive@tiers.org", w)

	if len(msg.Attachments) != 2 {
		t.Fatalf("attendu 2 pièces jointes (jeton + cert), reçu %d", len(msg.Attachments))
	}
	if msg.Attachments[0].Filename != "anchor-42.tsr" || string(msg.Attachments[0].Data) != string(w.Token) {
		t.Errorf("pièce jointe jeton incorrecte: %+v", msg.Attachments[0].Filename)
	}
	if msg.Attachments[1].Filename != "tsa-cert.pem" || string(msg.Attachments[1].Data) != string(w.TSACertPEM) {
		t.Errorf("pièce jointe cert incorrecte: %+v", msg.Attachments[1].Filename)
	}

	// Sans cert archivé, seul le jeton est joint.
	w.TSACertPEM = nil
	msg = buildWitnessMessage("noreply@exemple.fr", "archive@tiers.org", w)
	if len(msg.Attachments) != 1 {
		t.Fatalf("sans cert : attendu 1 pièce jointe, reçu %d", len(msg.Attachments))
	}
}

// TestBuildWitnessMessage_NoPII vérifie qu'aucune donnée personnelle ne sort :
// le message ne doit contenir que seq, hash, date et URL TSA.
func TestBuildWitnessMessage_NoPII(t *testing.T) {
	w := testWitness()
	msg := buildWitnessMessage("noreply@exemple.fr", "archive@tiers.org", w)

	full := msg.Subject + "\n" + msg.Body
	for _, forbidden := range []string{"user_id", "etudiant@", "nom", "prénom", "seance_id"} {
		if strings.Contains(strings.ToLower(full), forbidden) {
			t.Errorf("le témoin ne doit contenir aucune référence personnelle, trouvé %q", forbidden)
		}
	}
}

// ── Stub DBTX dédié au témoin ────────────────────────────────────────────────

// witnessRow simule pgx.Row pour les scans du témoin (bool, int64, string,
// time via pgtype non nécessaire ici car géré champ par champ).
type witnessRow struct {
	scan func(dest ...any) error
}

func (r *witnessRow) Scan(dest ...any) error { return r.scan(dest...) }

// witnessDB répond aux trois requêtes du flux témoin : GetAnchorByID,
// HasSentWitness, InsertWitness. Il mémorise les statuts insérés.
type witnessDB struct {
	sentByRecipient map[string]bool // état initial : recipients déjà SENT
	inserted        []string        // statuts insérés pendant le test
}

func (s *witnessDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (s *witnessDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query: " + sql)
}

func (s *witnessDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "FROM presence_anchor WHERE id"):
		return &witnessRow{scan: func(dest ...any) error {
			w := testWitness()
			*(dest[0].(*int64)) = w.AnchorID
			*(dest[1].(*int64)) = w.Seq
			*(dest[2].(*string)) = w.Hash
			*(dest[3].(*string)) = w.TSAURL
			*(dest[4].(*[]byte)) = w.Token
			*(dest[5].(*[]byte)) = w.TSACertPEM
			return nil // dest[6] (created_at pgtype.Timestamptz) laissé zéro
		}}
	case strings.Contains(sql, "FROM presence_witness"):
		recipient := args[1].(string)
		return &witnessRow{scan: func(dest ...any) error {
			*(dest[0].(*bool)) = s.sentByRecipient[recipient]
			return nil
		}}
	case strings.Contains(sql, "INSERT INTO presence_witness"):
		recipient := args[2].(string)
		status := args[3].(string)
		s.inserted = append(s.inserted, status)
		if status == "SENT" {
			s.sentByRecipient[recipient] = true
		}
		return &witnessRow{scan: func(dest ...any) error {
			*(dest[0].(*int64)) = int64(len(s.inserted))
			return nil
		}}
	}
	return &witnessRow{scan: func(...any) error { return errors.New("unexpected QueryRow: " + sql) }}
}

// ── Idempotence et envoi via SMTP local ──────────────────────────────────────

// TestSendWitnessForAnchor_Idempotent : deux passages sur le même anchor_id
// ne produisent qu'un seul SENT par destinataire.
func TestSendWitnessForAnchor_Idempotent(t *testing.T) {
	srv := startSMTPServer(t)
	cfg := services.WitnessConfig{
		Enabled:    true,
		Recipients: []string{"archive@tiers.org"},
		SMTP:       srv.config(),
	}
	db := &witnessDB{sentByRecipient: map[string]bool{}}

	report, err := SendWitnessForAnchor(context.Background(), db, cfg, 12)
	if err != nil {
		t.Fatal(err)
	}
	if report.Sent != 1 || report.Skipped != 0 || report.Failed != 0 {
		t.Fatalf("premier passage: attendu 1 SENT, reçu %+v", report)
	}

	report, err = SendWitnessForAnchor(context.Background(), db, cfg, 12)
	if err != nil {
		t.Fatal(err)
	}
	if report.Sent != 0 || report.Skipped != 1 {
		t.Fatalf("second passage: attendu 1 skip, reçu %+v", report)
	}
	if len(db.inserted) != 1 || db.inserted[0] != "SENT" {
		t.Fatalf("attendu exactement un SENT en base, reçu %v", db.inserted)
	}
	if len(srv.messages()) != 1 {
		t.Fatalf("attendu exactement 1 message SMTP, reçu %d", len(srv.messages()))
	}
}

// TestSendWitnessForAnchor_SMTPDown : un SMTP injoignable est tracé FAILED
// et ne remonte pas d'erreur (l'ancrage ne doit pas échouer).
func TestSendWitnessForAnchor_SMTPDown(t *testing.T) {
	cfg := services.WitnessConfig{
		Enabled:    true,
		Recipients: []string{"archive@tiers.org"},
		SMTP: services.SMTPConfig{
			Host: "127.0.0.1", Port: 1, // port fermé
			From: "noreply@exemple.fr", Timeout: 500 * time.Millisecond,
		},
	}
	db := &witnessDB{sentByRecipient: map[string]bool{}}

	report, err := SendWitnessForAnchor(context.Background(), db, cfg, 12)
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 1 || report.Sent != 0 {
		t.Fatalf("attendu 1 FAILED, reçu %+v", report)
	}
	if len(db.inserted) != 1 || db.inserted[0] != "FAILED" {
		t.Fatalf("attendu un FAILED en base, reçu %v", db.inserted)
	}
}

// TestSendWitnesses_OnlyNewAnchors : seules les ancres nouvellement créées
// déclenchent un témoin (pas de message si la tête n'a pas bougé).
func TestSendWitnesses_OnlyNewAnchors(t *testing.T) {
	srv := startSMTPServer(t)
	cfg := services.WitnessConfig{
		Enabled:    true,
		Recipients: []string{"archive@tiers.org"},
		SMTP:       srv.config(),
	}
	db := &witnessDB{sentByRecipient: map[string]bool{}}

	results := []AnchorResult{
		{TSAURL: "https://tsa-a", AnchorID: 12, Created: false},    // déjà existante
		{TSAURL: "https://tsa-b", AnchorID: 12, Created: true},     // nouvelle
		{TSAURL: "https://tsa-c", Err: errors.New("tsa http 500")}, // en échec
	}
	SendWitnesses(context.Background(), db, cfg, results)

	if got := len(srv.messages()); got != 1 {
		t.Fatalf("attendu 1 seul témoin (ancre nouvellement créée), reçu %d", got)
	}
}

// TestSendWitnesses_Disabled : témoin désactivé → aucun envoi, aucune erreur.
func TestSendWitnesses_Disabled(t *testing.T) {
	db := &witnessDB{sentByRecipient: map[string]bool{}}
	SendWitnesses(context.Background(), db, services.WitnessConfig{}, []AnchorResult{
		{AnchorID: 12, Created: true},
	})
	if len(db.inserted) != 0 {
		t.Fatalf("désactivé: aucune insertion attendue, reçu %v", db.inserted)
	}
}
