package presence

// Tests de la vérification d'un témoin externe. Aucun appel réseau : une
// mini-TSA locale (CA + certificat leaf ExtKeyUsageTimeStamping) forge des
// jetons RFC 3161 via digitorus/timestamp, et la base est simulée par un stub
// DBTX répondant à GetLedgerEntryByHash et VerifyChain.
//
// Fichier dans le package presence (et non presence_test) pour tester les
// helpers de décodage non exportés, comme witness_test.go.

import (
	"back-rex-common/pkg/ledger"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/digitorus/timestamp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ── Mini-TSA locale ──────────────────────────────────────────────────────────

type testTSA struct {
	caCert   *x509.Certificate
	caPEM    []byte
	leafCert *x509.Certificate
	leafKey  *ecdsa.PrivateKey
}

func newTestTSA(t *testing.T, name string) *testTSA {
	t.Helper()
	notBefore := time.Now().Add(-time.Hour)
	notAfter := time.Now().Add(24 * time.Hour)

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name + " Root CA"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: name + " TSA"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}

	return &testTSA{
		caCert:   caCert,
		caPEM:    pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}),
		leafCert: leafCert,
		leafKey:  leafKey,
	}
}

// token forge un jeton RFC 3161 scellant hashHex à la date when, signé par la
// TSA de test avec sa chaîne embarquée (comme un vrai .tsr de témoin).
func (tsa *testTSA) token(t *testing.T, hashHex string, when time.Time) []byte {
	t.Helper()
	hb, err := hex.DecodeString(hashHex)
	if err != nil {
		t.Fatal(err)
	}
	ts := &timestamp.Timestamp{
		HashAlgorithm:     crypto.SHA256,
		HashedMessage:     hb,
		Time:              when,
		Policy:            asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 999, 1},
		Certificates:      []*x509.Certificate{tsa.caCert},
		AddTSACertificate: true,
	}
	respDER, err := ts.CreateResponseWithOpts(tsa.leafCert, tsa.leafKey, crypto.SHA256)
	if err != nil {
		t.Fatalf("forge réponse TSA: %v", err)
	}
	parsed, err := timestamp.ParseResponse(respDER)
	if err != nil {
		t.Fatalf("parse réponse TSA forgée: %v", err)
	}
	return parsed.RawToken
}

// ── Stub DBTX : GetLedgerEntryByHash + VerifyChain ───────────────────────────

type vwEntry struct {
	Seq        int64
	SeanceID   int64
	UserID     int32
	Statut     string
	EventAt    time.Time
	RecordedAt time.Time
	PrevHash   string
	Hash       string
}

// buildVwChain construit n maillons chaînés valides en mémoire.
func buildVwChain(n int) []vwEntry {
	t0 := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	prevHash := ledger.GenesisHash
	entries := make([]vwEntry, 0, n)
	for i := 0; i < n; i++ {
		eventAt := t0.Add(time.Duration(i) * time.Minute)
		recordedAt := eventAt.Add(100 * time.Millisecond)
		e := vwEntry{
			Seq: int64(i + 1), SeanceID: 1, UserID: int32(i + 1), Statut: "PRESENT",
			EventAt: eventAt, RecordedAt: recordedAt, PrevHash: prevHash,
		}
		e.Hash = ledger.ComputeHash(ledger.LedgerEntry{
			SeanceID: e.SeanceID, UserID: e.UserID, Statut: e.Statut,
			EventAt: e.EventAt, RecordedAt: e.RecordedAt, PrevHash: e.PrevHash,
		})
		entries = append(entries, e)
		prevHash = e.Hash
	}
	return entries
}

func vwScan(dest any, v any) {
	switch d := dest.(type) {
	case *int64:
		*d = v.(int64)
	case *int32:
		*d = v.(int32)
	case *string:
		*d = v.(string)
	case *pgtype.Timestamptz:
		*d = pgtype.Timestamptz{Time: v.(time.Time), Valid: true}
	}
}

type vwRow struct {
	values []any
	err    error
}

func (r *vwRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if i >= len(r.values) {
			break
		}
		vwScan(d, r.values[i])
	}
	return nil
}

type vwRows struct {
	entries []vwEntry
	idx     int
}

func (r *vwRows) Next() bool                                   { r.idx++; return r.idx <= len(r.entries) }
func (r *vwRows) Close()                                       {}
func (r *vwRows) Err() error                                   { return nil }
func (r *vwRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *vwRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *vwRows) Values() ([]any, error)                       { return nil, nil }
func (r *vwRows) RawValues() [][]byte                          { return nil }
func (r *vwRows) Conn() *pgx.Conn                              { return nil }
func (r *vwRows) Scan(dest ...any) error {
	if r.idx < 1 || r.idx > len(r.entries) {
		return errors.New("out of range")
	}
	e := r.entries[r.idx-1]
	values := []any{e.Seq, e.SeanceID, e.UserID, e.Statut, e.EventAt, e.RecordedAt, e.PrevHash, e.Hash}
	for i, d := range dest {
		if i >= len(values) {
			break
		}
		vwScan(d, values[i])
	}
	return nil
}

// vwDB répond aux deux requêtes de VerifyWitness : la recherche par hash et le
// parcours ordonné de VerifyChain.
type vwDB struct {
	entries []vwEntry
}

func (s *vwDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("VerifyWitness doit être en lecture seule")
}

func (s *vwDB) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if strings.Contains(sql, "ListLedgerBySeq") {
		return &vwRows{entries: s.entries}, nil
	}
	return nil, errors.New("unexpected Query: " + sql)
}

func (s *vwDB) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if strings.Contains(sql, "FROM presence_ledger WHERE hash") {
		h := args[0].(string)
		for _, e := range s.entries {
			if e.Hash == h {
				return &vwRow{values: []any{e.Seq, e.EventAt, e.RecordedAt}}
			}
		}
		return &vwRow{err: pgx.ErrNoRows}
	}
	return &vwRow{err: errors.New("unexpected QueryRow: " + sql)}
}

// ── Verdicts ─────────────────────────────────────────────────────────────────

func TestVerifyWitness_Conforme(t *testing.T) {
	tsa := newTestTSA(t, "Alpha")
	entries := buildVwChain(3)
	sealed := time.Date(2026, 7, 9, 8, 0, 0, 0, time.UTC)
	token := tsa.token(t, entries[2].Hash, sealed)

	res, err := VerifyWitness(context.Background(), &vwDB{entries: entries}, token, tsa.caPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != WitnessConforme {
		t.Fatalf("attendu CONFORME, reçu %s (%s)", res.Verdict, res.Message)
	}
	if res.CoverageSeq != 3 {
		t.Errorf("coverage_seq attendu 3, reçu %d", res.CoverageSeq)
	}
	if res.SealedAt == nil || res.SealedAt.Unix() != sealed.Unix() {
		t.Errorf("sealed_at attendu %v, reçu %v", sealed, res.SealedAt)
	}
	if res.TSAName != "Alpha TSA" {
		t.Errorf("tsa_name attendu 'Alpha TSA', reçu %q", res.TSAName)
	}
	if res.HashHex != entries[2].Hash {
		t.Errorf("hash scellé attendu %s, reçu %s", entries[2].Hash, res.HashHex)
	}
}

// Un témoin intermédiaire (pas la tête actuelle) doit couvrir son maillon :
// la chaîne a grandi depuis le scellement, c'est le cas nominal.
func TestVerifyWitness_ConformeMaillonIntermediaire(t *testing.T) {
	tsa := newTestTSA(t, "Alpha")
	entries := buildVwChain(5)
	token := tsa.token(t, entries[1].Hash, time.Now().UTC())

	res, err := VerifyWitness(context.Background(), &vwDB{entries: entries}, token, tsa.caPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != WitnessConforme || res.CoverageSeq != 2 {
		t.Fatalf("attendu CONFORME/seq 2, reçu %s/seq %d", res.Verdict, res.CoverageSeq)
	}
}

func TestVerifyWitness_ReecritureDetectee(t *testing.T) {
	tsa := newTestTSA(t, "Alpha")
	entries := buildVwChain(3)
	// Hash scellé à l'époque, absent de la chaîne actuelle (réécrite).
	disparu := strings.Repeat("ab", 32)
	token := tsa.token(t, disparu, time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC))

	res, err := VerifyWitness(context.Background(), &vwDB{entries: entries}, token, tsa.caPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != WitnessReecritureDetectee {
		t.Fatalf("attendu REECRITURE_DETECTEE, reçu %s (%s)", res.Verdict, res.Message)
	}
	if res.SealedAt == nil {
		t.Error("sealed_at doit être fourni même en cas de réécriture (dichotomie)")
	}
}

func TestVerifyWitness_ChaineCorrompue(t *testing.T) {
	tsa := newTestTSA(t, "Alpha")
	entries := buildVwChain(3)
	// Altération du maillon 2 sans recalcul du hash : le hash de tête (seq 3)
	// existe toujours, mais VerifyChain casse à 2 ≤ 3.
	entries[1].Statut = "ABSENT"
	token := tsa.token(t, entries[2].Hash, time.Now().UTC())

	res, err := VerifyWitness(context.Background(), &vwDB{entries: entries}, token, tsa.caPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != WitnessChaineCorrompue {
		t.Fatalf("attendu CHAINE_CORROMPUE, reçu %s (%s)", res.Verdict, res.Message)
	}
	if res.BrokenSeq != 2 {
		t.Errorf("seq de rupture attendu 2, reçu %d", res.BrokenSeq)
	}
	if res.CoverageSeq != 3 {
		t.Errorf("le maillon trouvé (3) doit être rapporté, reçu %d", res.CoverageSeq)
	}
}

// Une rupture APRÈS le maillon couvert ne remet pas en cause le témoin : la
// partie scellée est conforme, la rupture est signalée dans le message.
func TestVerifyWitness_RuptureApresCouverture(t *testing.T) {
	tsa := newTestTSA(t, "Alpha")
	entries := buildVwChain(5)
	entries[4].Statut = "ABSENT" // rupture à seq 5
	token := tsa.token(t, entries[1].Hash, time.Now().UTC())

	res, err := VerifyWitness(context.Background(), &vwDB{entries: entries}, token, tsa.caPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != WitnessConforme || res.CoverageSeq != 2 {
		t.Fatalf("attendu CONFORME jusqu'à 2, reçu %s/seq %d", res.Verdict, res.CoverageSeq)
	}
	if !strings.Contains(res.Message, "5") {
		t.Errorf("le message doit signaler la rupture ultérieure (seq 5): %s", res.Message)
	}
}

func TestVerifyWitness_TokenInvalide(t *testing.T) {
	res, err := VerifyWitness(context.Background(), &vwDB{}, []byte("pas un jeton DER"), nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != WitnessTokenInvalide {
		t.Fatalf("attendu TOKEN_INVALIDE, reçu %s", res.Verdict)
	}
}

func TestVerifyWitness_SignatureInvalide(t *testing.T) {
	tsaAlpha := newTestTSA(t, "Alpha")
	tsaBravo := newTestTSA(t, "Bravo")
	entries := buildVwChain(3)
	token := tsaAlpha.token(t, entries[2].Hash, time.Now().UTC())

	// L'auditeur colle le certificat d'une AUTRE autorité : la chaîne de
	// certification ne remonte pas, le témoin n'est pas probant.
	res, err := VerifyWitness(context.Background(), &vwDB{entries: entries}, token, tsaBravo.caPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != WitnessSignatureInvalide {
		t.Fatalf("attendu SIGNATURE_INVALIDE, reçu %s (%s)", res.Verdict, res.Message)
	}
}

// ── Décodage des entrées collées par l'auditeur ──────────────────────────────

func TestDecodeWitnessToken_Formats(t *testing.T) {
	der := []byte{0x30, 0x82, 0x01, 0x00, 0xde, 0xad, 0xbe, 0xef}

	// DER brut (fichier .tsr téléversé).
	if got, err := decodeWitnessToken(der); err != nil || string(got) != string(der) {
		t.Errorf("DER brut: %v", err)
	}

	// Base64 avec sauts de ligne (copier-coller d'e-mail).
	b64 := base64.StdEncoding.EncodeToString(der)
	wrapped := "  " + b64[:6] + "\n" + b64[6:] + "\r\n"
	if got, err := decodeWitnessToken([]byte(wrapped)); err != nil || string(got) != string(der) {
		t.Errorf("base64 replié: %v", err)
	}

	// PEM, quel que soit le type de bloc.
	pemTxt := pem.EncodeToMemory(&pem.Block{Type: "TIMESTAMP TOKEN", Bytes: der})
	if got, err := decodeWitnessToken(pemTxt); err != nil || string(got) != string(der) {
		t.Errorf("PEM: %v", err)
	}

	// Erreurs de format : message clair, pas de panique.
	for name, input := range map[string]string{
		"vide":        "   \n",
		"texte":       "ceci n'est pas un jeton !!",
		"pem tronqué": "-----BEGIN TIMESTAMP TOKEN-----\nMIIC\n",
	} {
		if _, err := decodeWitnessToken([]byte(input)); err == nil {
			t.Errorf("%s: erreur attendue", name)
		}
	}
}

func TestDecodeWitnessCert(t *testing.T) {
	tsa := newTestTSA(t, "Alpha")

	if got, err := decodeWitnessCert(""); err != nil || got != nil {
		t.Errorf("cert vide → nil sans erreur, reçu %v/%v", got, err)
	}
	if _, err := decodeWitnessCert("pas du PEM"); err == nil {
		t.Error("cert non PEM: erreur attendue")
	}
	if got, err := decodeWitnessCert(string(tsa.caPEM)); err != nil || len(got) == 0 {
		t.Errorf("cert PEM valide refusé: %v", err)
	}
}
