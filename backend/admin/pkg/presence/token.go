package presence

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var tokenSecret string

func SetTokenSecret(secret string) {
	tokenSecret = secret
}

type tokenPayload struct {
	SeanceID int64 `json:"seance_id"`
	Exp      int64 `json:"exp"`
}

const tokenTTL = 30 * time.Second

func IssueToken(seanceID int64) string {
	payload := tokenPayload{
		SeanceID: seanceID,
		Exp:      time.Now().Add(tokenTTL).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	mac := hmac.New(sha256.New, []byte(tokenSecret))
	mac.Write(payloadJSON)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payloadB64 + "." + sig
}

func VerifyToken(token string) (int64, error) {
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
