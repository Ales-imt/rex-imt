package bullettin

// Conversion docx -> PDF déléguée à un service Gotenberg (sidecar).
// Gotenberg v8 : POST {base}/forms/libreoffice/convert, champ multipart
// « files » (un seul fichier -> le corps de la réponse est le PDF).

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// convertirDocxEnPDF envoie docxPath à Gotenberg et écrit le PDF dans pdfPath.
func convertirDocxEnPDF(ctx context.Context, gotenbergURL, docxPath, pdfPath string) error {
	docx, err := os.Open(docxPath)
	if err != nil {
		return err
	}
	defer docx.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("files", filepath.Base(docxPath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, docx); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}

	url := strings.TrimRight(gotenbergURL, "/") + "/forms/libreoffice/convert"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	// Gotenberg est un sidecar local : ne jamais router l'appel via un proxy.
	// Le transport par défaut honore HTTP_PROXY/NO_PROXY de l'environnement, ce
	// qui enverrait la requête vers le proxy SOCKS (réseau école) au lieu du
	// conteneur, d'où un blocage puis un 504 en amont.
	client := &http.Client{
		Timeout:   2 * time.Minute,
		Transport: &http.Transport{Proxy: nil},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return fmt.Errorf("gotenberg a répondu %s : %s", resp.Status, strings.TrimSpace(string(msg)))
	}

	pdf, err := os.Create(pdfPath)
	if err != nil {
		return err
	}
	defer pdf.Close()

	if _, err := io.Copy(pdf, resp.Body); err != nil {
		return err
	}
	return nil
}
