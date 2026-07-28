package service

import (
	"bytes"
	"fmt"
	"net/http"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// encryptStrongbox chiffre l'IP, l'ID élève et son nom/prénom avec la clef publique age fournie.
func EncryptStrongbox(publicKeyStr string, r *http.Request, studentID int, nom string, prenom string) (string, error) {
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

	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		return "", fmt.Errorf("le message doit contenit une IP dans le header X-Real-IP")
	}

	payload := fmt.Sprintf("ip=%s student_id=%d nom=%s prenom=%s", ip, studentID, nom, prenom)
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
