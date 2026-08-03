package services

import (
	"bytes"
	"fmt"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// EncryptAge chiffre un texte avec la clef publique age (X25519) fournie et
// renvoie le résultat armuré (ASCII). Utilisé pour mettre au coffre un contenu
// qui ne doit plus être lisible en clair au repos (ex. le texte d'un feedback
// refusé, conservé chiffré le temps d'une éventuelle contestation puis purgé).
func EncryptAge(publicKeyStr string, plaintext string) (string, error) {
	recipient, err := age.ParseX25519Recipient(publicKeyStr)
	if err != nil {
		return "", fmt.Errorf("clef publique age invalide : %w", err)
	}

	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)
	w, err := age.Encrypt(armorWriter, recipient)
	if err != nil {
		return "", fmt.Errorf("age encrypt : %w", err)
	}

	if _, err := w.Write([]byte(plaintext)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	if err := armorWriter.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
