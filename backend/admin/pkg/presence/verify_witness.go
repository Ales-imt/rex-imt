package presence

// Vérification d'un témoin externe — l'auditeur colle le jeton RFC 3161 reçu
// par e-mail depuis la boîte externe (voir witness.go et docs/temoin-externe.md)
// et on le confronte à la base :
//
//  1. le jeton, signature validée, prouve qu'un hash de tête H existait à une
//     date T certifiée par la TSA ;
//  2. si H figure dans presence_ledger.hash au maillon N, la chaîne actuelle
//     reproduit l'état scellé → conforme jusqu'à N ; s'il en est absent, la
//     chaîne a été réécrite après T ;
//  3. VerifyChain confirme en complément la cohérence interne jusqu'à N.
//
// La preuve vient de la confrontation de deux sources indépendantes : le témoin
// détenu par un tiers et la base. La vérification est strictement en lecture.
//
// Le motif de vérification du jeton (Parse + HashedMessage + CertPool) est
// celui de VerifyAnchors (anchor.go).

import (
	"back-rex-common/pkg/ledger"
	ledgergen "back-rex-common/pkg/ledger/gen"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/digitorus/timestamp"
	"github.com/jackc/pgx/v5"
)

// WitnessVerdict est le verdict de la confrontation témoin ↔ base.
type WitnessVerdict string

const (
	// WitnessConforme : le hash scellé existe dans la chaîne et le chaînage
	// interne est valide au moins jusqu'à lui.
	WitnessConforme WitnessVerdict = "CONFORME"
	// WitnessReecritureDetectee : le hash certifié par la TSA est absent de la
	// chaîne actuelle — les données ont été réécrites après la date scellée.
	WitnessReecritureDetectee WitnessVerdict = "REECRITURE_DETECTEE"
	// WitnessChaineCorrompue : le hash scellé est présent mais le chaînage
	// interne est rompu avant lui.
	WitnessChaineCorrompue WitnessVerdict = "CHAINE_CORROMPUE"
	// WitnessTokenInvalide : le jeton est illisible (ASN.1/CMS invalide).
	WitnessTokenInvalide WitnessVerdict = "TOKEN_INVALIDE"
	// WitnessSignatureInvalide : la chaîne de certification TSA ne remonte pas
	// au certificat fourni — le témoin n'est pas probant.
	WitnessSignatureInvalide WitnessVerdict = "SIGNATURE_INVALIDE"
)

// WitnessResult est le résultat de VerifyWitness.
type WitnessResult struct {
	Verdict WitnessVerdict `json:"verdict"`
	// SealedAt est la date certifiée par la TSA (ts.Time), dès que le jeton est
	// lisible — y compris pour un verdict négatif, afin de situer le test sur
	// la frise temporelle (dichotomie manuelle).
	SealedAt *time.Time `json:"sealedAt,omitempty"`
	// CoverageSeq est le maillon où le hash scellé a été trouvé : la chaîne est
	// authentifiée jusqu'à lui inclus (verdict CONFORME).
	CoverageSeq int64 `json:"coverageSeq,omitempty"`
	// BrokenSeq est le maillon de rupture du chaînage interne (CHAINE_CORROMPUE).
	BrokenSeq int64 `json:"brokenSeq,omitempty"`
	// TSAName identifie l'autorité d'horodatage (CN du certificat signataire).
	TSAName string `json:"tsaName,omitempty"`
	// HashHex est l'empreinte scellée par le jeton (hex).
	HashHex string `json:"hashHex,omitempty"`
	Message string `json:"message"`
}

// VerifyWitness confronte un jeton RFC 3161 fourni par l'auditeur à l'état
// actuel de presence_ledger. tsaCertPEM est le certificat TSA collé par
// l'auditeur (optionnel) ; à défaut, le CA racine caCertPath de la config sert
// de point de confiance. Lecture seule : aucune écriture en base.
func VerifyWitness(ctx context.Context, db DBTX, tokenDER []byte, tsaCertPEM []byte, caCertPath string) (WitnessResult, error) {
	// 1. Parse vérifie l'ASN.1 et la signature CMS contre le cert embarqué
	//    (motif VerifyAnchors). Tolérance : certains .tsr contiennent la
	//    réponse TSA complète plutôt que le jeton nu.
	ts, parseErr := timestamp.Parse(tokenDER)
	if parseErr != nil {
		if tsResp, respErr := timestamp.ParseResponse(tokenDER); respErr == nil {
			ts = tsResp
		} else {
			return WitnessResult{
				Verdict: WitnessTokenInvalide,
				Message: fmt.Sprintf("Jeton illisible (ni jeton RFC 3161, ni réponse TSA) : %v", parseErr),
			}, nil
		}
	}

	sealedAt := ts.Time
	res := WitnessResult{
		SealedAt: &sealedAt,
		HashHex:  hex.EncodeToString(ts.HashedMessage),
	}
	if len(ts.Certificates) > 0 {
		res.TSAName = ts.Certificates[0].Subject.CommonName
	}

	// 2. Chaîne de certification TSA : cert collé par l'auditeur, sinon CA
	//    racine de la config (même motif optionnel que VerifyAnchors).
	var rootCAPEM []byte
	if caCertPath != "" {
		var readErr error
		rootCAPEM, readErr = os.ReadFile(caCertPath)
		if readErr != nil {
			return WitnessResult{}, fmt.Errorf("read ca cert %q: %w", caCertPath, readErr)
		}
	}
	if len(ts.Certificates) > 0 && (len(tsaCertPEM) > 0 || len(rootCAPEM) > 0) {
		pool := x509.NewCertPool()
		if len(tsaCertPEM) > 0 {
			pool.AppendCertsFromPEM(tsaCertPEM)
		}
		if len(rootCAPEM) > 0 {
			pool.AppendCertsFromPEM(rootCAPEM)
		}
		// KeyUsages est indispensable : sans lui, x509.Verify exige l'EKU
		// ServerAuth alors qu'un certificat TSA porte l'EKU TimeStamping
		// (RFC 3161 §2.3).
		opts := x509.VerifyOptions{
			Roots:     pool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
		}
		if _, certErr := ts.Certificates[0].Verify(opts); certErr != nil {
			res.Verdict = WitnessSignatureInvalide
			res.Message = fmt.Sprintf("La chaîne de certification TSA ne remonte pas au certificat de confiance fourni : %v. Ce témoin n'est pas probant.", certErr)
			return res, nil
		}
	}

	// 3. Le hash scellé existe-t-il encore dans la chaîne actuelle ?
	entry, err := ledgergen.New(db).GetLedgerEntryByHash(ctx, res.HashHex)
	if errors.Is(err, pgx.ErrNoRows) {
		res.Verdict = WitnessReecritureDetectee
		res.Message = fmt.Sprintf("Le hash certifié le %s est absent de la chaîne actuelle : les données ont été modifiées après cette date.",
			sealedAt.UTC().Format(time.RFC3339))
		return res, nil
	}
	if err != nil {
		return WitnessResult{}, fmt.Errorf("witness lookup hash: %w", err)
	}
	res.CoverageSeq = entry.Seq

	// 4. Cohérence interne du chaînage jusqu'au maillon couvert.
	chain, err := ledger.VerifyChain(ctx, db)
	if err != nil {
		return WitnessResult{}, fmt.Errorf("witness verify chain: %w", err)
	}
	if !chain.OK && chain.BrokenAt <= entry.Seq {
		res.Verdict = WitnessChaineCorrompue
		res.BrokenSeq = chain.BrokenAt
		res.Message = fmt.Sprintf("Le hash scellé le %s figure au maillon %d, mais le chaînage interne est rompu au maillon %d : %s",
			sealedAt.UTC().Format(time.RFC3339), entry.Seq, chain.BrokenAt, chain.Error)
		return res, nil
	}

	res.Verdict = WitnessConforme
	res.Message = fmt.Sprintf("Chaîne conforme jusqu'au maillon %d, scellé le %s%s. Tout ce qui précède ce point est authentifié.",
		entry.Seq, sealedAt.UTC().Format(time.RFC3339), tsaSuffix(res.TSAName))
	if !chain.OK {
		res.Message += fmt.Sprintf(" Attention : le chaînage interne est rompu plus loin (maillon %d), au-delà de la couverture de ce témoin.", chain.BrokenAt)
	}
	return res, nil
}

func tsaSuffix(name string) string {
	if name == "" {
		return ""
	}
	return " par " + name
}

// decodeWitnessToken décode le jeton collé ou téléversé par l'auditeur, avec
// tolérance sur le format (copier-coller depuis un e-mail) : PEM (tout type de
// bloc), base64 avec sauts de ligne, ou DER brut. Le base64 est tenté AVANT le
// DER : 0x30 est aussi le caractère '0', qu'un jeton en base64 peut ouvrir.
func decodeWitnessToken(raw []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, errors.New("aucun témoin fourni : collez le jeton reçu par e-mail ou téléversez le fichier .tsr")
	}

	if strings.Contains(trimmed, "-----BEGIN") {
		block, _ := pem.Decode([]byte(trimmed))
		if block == nil {
			return nil, errors.New("bloc PEM illisible : vérifiez que le témoin est copié en entier, en-têtes -----BEGIN/END compris")
		}
		return block.Bytes, nil
	}

	// Base64, en tolérant espaces et sauts de ligne introduits par l'e-mail.
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, trimmed)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(compact); err == nil && len(b) > 0 {
			return b, nil
		}
	}

	// Fichier .tsr binaire : une structure DER commence par 0x30 (SEQUENCE).
	if raw[0] == 0x30 {
		return raw, nil
	}

	return nil, errors.New("format non reconnu : collez le jeton en base64 ou PEM, ou téléversez le fichier .tsr tel quel")
}

// decodeWitnessCert valide le certificat TSA optionnel collé par l'auditeur.
// Chaîne vide → nil (fallback sur le CA racine de la config).
func decodeWitnessCert(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if !strings.Contains(trimmed, "-----BEGIN CERTIFICATE-----") {
		return nil, errors.New("certificat TSA invalide : attendu au format PEM (-----BEGIN CERTIFICATE-----)")
	}
	if block, _ := pem.Decode([]byte(trimmed)); block == nil {
		return nil, errors.New("certificat TSA illisible : vérifiez que le PEM est copié en entier")
	}
	return []byte(trimmed), nil
}
