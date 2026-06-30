package presence

// Ancrage RFC 3161 — scelle le dernier hash du registre auprès d'une TSA externe.
//
// Seul le hash transite vers la TSA : aucune donnée personnelle ne quitte le
// système (avantage RGPD). Le jeton retourné et le certificat TSA sont archivés
// dans presence_anchor pour permettre une vérification future même si la TSA
// disparaît.
//
// Dépendance : github.com/digitorus/timestamp (déjà dans go.mod)

import (
	"back-rex-common/pkg/services"
	"bytes"
	"context"
	"crypto"
	_ "crypto/sha256" // enregistre SHA-256 dans le registre de hashes
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/digitorus/timestamp"
	"github.com/jackc/pgx/v5"
)

// AnchorResult est retourné par AnchorLast pour chaque TSA.
type AnchorResult struct {
	TSAURL   string
	AnchorID int64
	Err      error
}

// AnchorLast prend le dernier maillon de presence_ledger, demande un horodatage
// RFC 3161 à chaque TSA configurée, et archive les jetons dans presence_anchor.
//
// Idempotente par (ledger_seq, tsa_url) : si une ancre existe déjà pour le
// dernier maillon et une TSA donnée, cette TSA est ignorée.
//
// RGPD : seul le hash SHA-256 est envoyé à la TSA ; aucune donnée personnelle
// ne quitte le système.
func AnchorLast(ctx context.Context, db DBTX, cfg services.TimestampConfig) ([]AnchorResult, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	urls := cfg.URLs
	if len(urls) == 0 {
		urls = []string{services.DefaultTSAURL}
	}

	seq, h, err := GetLastLedgerEntry(ctx, db)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // ledger vide, rien à ancrer
	}
	if err != nil {
		return nil, fmt.Errorf("anchor get last entry: %w", err)
	}

	// hashBytes = 32 octets du SHA-256 du maillon. C'est CE hash que la TSA va
	// horodater (MessageImprint). Aucune donnée personnelle n'en est déductible.
	hashBytes, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("anchor decode hash: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	httpClient := &http.Client{Timeout: timeout}

	algo := cfg.HashAlgorithm
	if algo == "" {
		algo = "sha256"
	}

	results := make([]AnchorResult, 0, len(urls))

	for _, tsaURL := range urls {
		res := AnchorResult{TSAURL: tsaURL}

		// Idempotence : skip si ancre déjà existante pour ce (seq, tsaURL).
		var existingID int64
		scanErr := db.QueryRow(ctx,
			"SELECT id FROM presence_anchor WHERE ledger_seq = $1 AND tsa_url = $2 LIMIT 1",
			seq, tsaURL,
		).Scan(&existingID)
		if scanErr == nil {
			res.AnchorID = existingID
			results = append(results, res)
			continue
		}
		if !errors.Is(scanErr, pgx.ErrNoRows) {
			res.Err = fmt.Errorf("check existing anchor: %w", scanErr)
			results = append(results, res)
			continue
		}

		// Construction de la requête TSA : on passe le hash directement dans
		// HashedMessage sans le re-hasher (Request.Marshal est plus bas niveau
		// que CreateRequest qui hashe lui-même le contenu d'un io.Reader).
		req := &timestamp.Request{
			HashAlgorithm: crypto.SHA256,
			HashedMessage: hashBytes,
			Certificates:  true,
		}
		tsaReqBytes, err := req.Marshal()
		if err != nil {
			res.Err = fmt.Errorf("marshal tsa request: %w", err)
			results = append(results, res)
			continue
		}

		httpResp, err := httpClient.Post(tsaURL, "application/timestamp-query", bytes.NewReader(tsaReqBytes))
		if err != nil {
			res.Err = fmt.Errorf("tsa post: %w", err)
			results = append(results, res)
			continue
		}
		respBody, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			res.Err = fmt.Errorf("tsa read: %w", err)
			results = append(results, res)
			continue
		}
		if httpResp.StatusCode != http.StatusOK {
			res.Err = fmt.Errorf("tsa http %d", httpResp.StatusCode)
			results = append(results, res)
			continue
		}

		// ParseResponse vérifie la signature CMS si le certificat est inclus.
		tsResp, err := timestamp.ParseResponse(respBody)
		if err != nil {
			res.Err = fmt.Errorf("parse tsa response: %w", err)
			results = append(results, res)
			continue
		}

		// Vérification supplémentaire : le hash horodaté correspond bien au nôtre.
		if !bytes.Equal(tsResp.HashedMessage, hashBytes) {
			res.Err = fmt.Errorf("tsa response hash mismatch")
			results = append(results, res)
			continue
		}

		// Sérialisation des certificats TSA en PEM pour archivage durable.
		// Archiver les certs AVEC le jeton garantit la vérifiabilité future
		// même si la TSA disparaît ou révoque ses clés.
		var tsaCertPEM []byte
		for _, cert := range tsResp.Certificates {
			tsaCertPEM = append(tsaCertPEM, pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert.Raw,
			})...)
		}

		var anchorID int64
		err = db.QueryRow(ctx,
			`INSERT INTO presence_anchor
			 (ledger_seq, anchored_hash, tsa_url, hash_algorithm, token, tsa_cert)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id`,
			seq, h, tsaURL, algo, tsResp.RawToken, tsaCertPEM,
		).Scan(&anchorID)
		if err != nil {
			res.Err = fmt.Errorf("insert anchor: %w", err)
			results = append(results, res)
			continue
		}
		res.AnchorID = anchorID
		results = append(results, res)
	}
	return results, nil
}

// AnchorVerifyResult est le résultat de la vérification d'une ligne presence_anchor.
type AnchorVerifyResult struct {
	AnchorID  int64
	LedgerSeq int64
	TSAURL    string
	OK        bool
	Err       string
}

// VerifyAnchors lit toutes les lignes de presence_anchor et vérifie chaque jeton
// RFC 3161.
//
// Vérification en deux étapes :
//  1. timestamp.Parse vérifie la signature CMS du jeton contre le cert embarqué.
//  2. On compare HashedMessage au anchored_hash stocké en base.
//
// Si caCertPath est fourni, le certificat racine de la TSA est chargé et ajouté
// au pool de confiance, permettant la vérification hors ligne (sans OCSP/CRL).
func VerifyAnchors(ctx context.Context, db DBTX, caCertPath string) ([]AnchorVerifyResult, error) {
	var rootCAPEM []byte
	if caCertPath != "" {
		var readErr error
		rootCAPEM, readErr = os.ReadFile(caCertPath)
		if readErr != nil {
			return nil, fmt.Errorf("read ca cert %q: %w", caCertPath, readErr)
		}
	}

	rows, err := db.Query(ctx,
		`SELECT id, ledger_seq, anchored_hash, tsa_url, token, tsa_cert
		 FROM presence_anchor ORDER BY ledger_seq ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list anchors: %w", err)
	}
	defer rows.Close()

	var results []AnchorVerifyResult
	for rows.Next() {
		var (
			anchorID     int64
			ledgerSeq    int64
			anchoredHash string
			tsaURL       string
			token        []byte
			tsaCertPEM   []byte
		)
		if err := rows.Scan(&anchorID, &ledgerSeq, &anchoredHash, &tsaURL, &token, &tsaCertPEM); err != nil {
			return nil, fmt.Errorf("scan anchor: %w", err)
		}

		res := AnchorVerifyResult{AnchorID: anchorID, LedgerSeq: ledgerSeq, TSAURL: tsaURL}

		hashBytes, decErr := hex.DecodeString(anchoredHash)
		if decErr != nil {
			res.Err = fmt.Sprintf("decode anchored_hash: %v", decErr)
			results = append(results, res)
			continue
		}

		// Parse vérifie la signature CMS (si le jeton contient son certificat).
		ts, parseErr := timestamp.Parse(token)
		if parseErr != nil {
			res.Err = fmt.Sprintf("parse token: %v", parseErr)
			results = append(results, res)
			continue
		}

		// Vérification que le hash horodaté correspond au hash stocké en base.
		if !bytes.Equal(ts.HashedMessage, hashBytes) {
			res.Err = "token HashedMessage does not match anchored_hash"
			results = append(results, res)
			continue
		}

		// Vérification optionnelle de la chaîne de certification.
		if len(ts.Certificates) > 0 && (len(tsaCertPEM) > 0 || len(rootCAPEM) > 0) {
			pool := x509.NewCertPool()
			if len(tsaCertPEM) > 0 {
				pool.AppendCertsFromPEM(tsaCertPEM)
			}
			if len(rootCAPEM) > 0 {
				pool.AppendCertsFromPEM(rootCAPEM)
			}
			_, certErr := ts.Certificates[0].Verify(x509.VerifyOptions{Roots: pool})
			if certErr != nil {
				res.Err = fmt.Sprintf("cert chain: %v", certErr)
				results = append(results, res)
				continue
			}
		}

		res.OK = true
		results = append(results, res)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("anchors rows: %w", err)
	}
	return results, nil
}
