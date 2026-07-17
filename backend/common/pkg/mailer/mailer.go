// Package mailer fournit un envoi SMTP minimal (stdlib uniquement) avec
// pièces jointes. Conçu pour un usage « send-only » : le compte SMTP utilisé
// ne doit avoir aucun droit de lecture sur la boîte destinataire.
package mailer

import (
	"back-rex-common/pkg/services"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// Attachment est une pièce jointe binaire encodée en base64 à l'envoi.
type Attachment struct {
	Filename    string
	ContentType string // ex. "application/timestamp-reply", "application/x-pem-file"
	Data        []byte
}

// Message est un e-mail texte avec pièces jointes optionnelles.
type Message struct {
	From        string
	To          []string
	Subject     string
	Body        string // text/plain UTF-8
	Attachments []Attachment
}

const defaultTimeout = 10 * time.Second

// Send transmet le message via le serveur SMTP configuré.
// STARTTLS et l'authentification PLAIN sont appliqués selon cfg.
// Toutes les erreurs sont enveloppées ; aucune panic.
func Send(ctx context.Context, cfg services.SMTPConfig, msg Message) error {
	if cfg.Host == "" {
		return fmt.Errorf("smtp send: host non configuré")
	}
	if len(msg.To) == 0 {
		return fmt.Errorf("smtp send: aucun destinataire")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	// Deadline globale : couvre toute la session SMTP, pas seulement le dial.
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		conn.Close()
		return fmt.Errorf("smtp set deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if cfg.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp: STARTTLS requis par la config mais non proposé par %s", cfg.Host)
		}
		if err := client.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if cfg.Username != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(cfg.From); err != nil {
		return fmt.Errorf("smtp mail from %q: %w", cfg.From, err)
	}
	for _, rcpt := range msg.To {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %q: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	raw, err := buildMIME(cfg.From, msg)
	if err != nil {
		return fmt.Errorf("smtp build mime: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

// buildMIME assemble le message au format multipart/mixed (corps texte +
// pièces jointes base64). Sans pièce jointe, un simple text/plain est produit.
func buildMIME(from string, msg Message) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(msg.To, ", "))
	fmt.Fprintf(&buf, "Subject: %s\r\n", msg.Subject)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(msg.Attachments) == 0 {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		buf.WriteString(msg.Body)
		return buf.Bytes(), nil
	}

	mw := multipart.NewWriter(&buf)
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", mw.Boundary())

	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	part, err := mw.CreatePart(textHeader)
	if err != nil {
		return nil, fmt.Errorf("mime text part: %w", err)
	}
	if _, err := part.Write([]byte(msg.Body)); err != nil {
		return nil, fmt.Errorf("mime write text: %w", err)
	}

	for _, att := range msg.Attachments {
		h := textproto.MIMEHeader{}
		contentType := att.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		h.Set("Content-Type", contentType)
		h.Set("Content-Transfer-Encoding", "base64")
		h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))
		part, err := mw.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("mime part %q: %w", att.Filename, err)
		}
		if err := writeBase64(part, att.Data); err != nil {
			return nil, fmt.Errorf("mime encode %q: %w", att.Filename, err)
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("mime close: %w", err)
	}
	return buf.Bytes(), nil
}

// writeBase64 encode data en base64 avec des lignes de 76 caractères (RFC 2045).
func writeBase64(w interface{ Write([]byte) (int, error) }, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 0 {
		n := 76
		if len(encoded) < n {
			n = len(encoded)
		}
		if _, err := w.Write([]byte(encoded[:n] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[n:]
	}
	return nil
}
