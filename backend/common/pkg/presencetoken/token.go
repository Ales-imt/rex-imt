package presencetoken

// Token de séance HMAC — émis par le service qui affiche le QR (admin,
// student côté prof) et vérifié par le pointage étudiant. UNIQUE
// implémentation, partagée pour que signature et format ne divergent jamais.
//
// Format : base64url(JSON{seance_id,exp}).base64url(HMAC-SHA256)

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"
)

// codeAlphabet exclut les caractères ambigus (I, L, O, 0, 1) pour une saisie
// manuelle fiable du code de repli affiché à côté du QR.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// GenerateCode produit le code court assigné à une séance à son ouverture.
func GenerateCode() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = codeAlphabet[rand.Intn(len(codeAlphabet))]
	}
	return string(b)
}

var tokenSecret string

// SetSecret installe le secret HMAC partagé (config presence.tokenSecret,
// identique entre les services pour qu'un token émis ici se vérifie là-bas).
func SetSecret(secret string) {
	tokenSecret = secret
}

type tokenPayload struct {
	SeanceID int64 `json:"seance_id"`
	Exp      int64 `json:"exp"`
}

// TTL est la durée de validité d'un token (le QR se rafraîchit avant expiration).
const TTL = 30 * time.Second

func Issue(seanceID int64) string {
	payload := tokenPayload{
		SeanceID: seanceID,
		Exp:      time.Now().Add(TTL).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	mac := hmac.New(sha256.New, []byte(tokenSecret))
	mac.Write(payloadJSON)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payloadB64 + "." + sig
}

func Verify(token string) (int64, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return 0, errors.New("token invalide")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, errors.New("token invalide")
	}

	mac := hmac.New(sha256.New, []byte(tokenSecret))
	mac.Write(payloadJSON)
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return 0, errors.New("signature invalide")
	}

	var p tokenPayload
	if err := json.Unmarshal(payloadJSON, &p); err != nil {
		return 0, errors.New("token invalide")
	}
	if time.Now().Unix() > p.Exp {
		return 0, errors.New("token expiré")
	}
	return p.SeanceID, nil
}
