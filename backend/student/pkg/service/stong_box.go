package service

import (
	"bytes"
	"fmt"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// encryptStrongbox chiffre l'IP et l'ID élève avec la clef publique age fournie.
func EncryptStrongbox(publicKeyStr string, ip string, studentID int) (string, error) {
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

	payload := fmt.Sprintf("ip=%s student_id=%d", ip, studentID)
	if _, err := w.Write([]byte(payload)); err != nil {
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
