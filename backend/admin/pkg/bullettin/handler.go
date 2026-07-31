package bullettin

// Le front renvoie le .xlsx (et le .docx template) au démarrage d'une
// génération. Les fichiers reçus sont écrits dans des fichiers temporaires le
// temps du traitement, puis nettoyés. La génération est asynchrone (job +
// polling), voir jobs.go.

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// saveUpload écrit un fichier uploadé (champ multipart) dans un fichier
// temporaire portant le même suffixe, et renvoie son chemin. L'appelant doit
// faire defer os.Remove(path).
func saveUpload(r *http.Request, field string) (string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return "", fmt.Errorf("champ %q manquant : %w", field, err)
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "upload-*"+filepath.Ext(header.Filename))
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, file); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, err.Error())
}

func (h *handlers) handleSheets(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	xlsxPath, err := saveUpload(r, "xlsx")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	defer os.Remove(xlsxPath)

	book, err := ouvrirClasseur(xlsxPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, book.order)
}

func (h *handlers) handleColumns(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	xlsxPath, err := saveUpload(r, "xlsx")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	defer os.Remove(xlsxPath)

	sheetName := strings.TrimSpace(r.FormValue("sheet"))

	book, err := ouvrirClasseur(xlsxPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	f, ok := book.sheets[sheetName]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("feuille introuvable : %q", sheetName))
		return
	}

	writeJSON(w, listerColonnes(f))
}

// handleGenerateStart valide les entrées, démarre la génération en arrière-plan
// et renvoie immédiatement l'identifiant du job ({"id": "..."}). Le front suit
// l'avancement via /generate/{id}/status puis récupère le zip via /download.
func (h *handlers) handleGenerateStart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	xlsxPath, err := saveUpload(r, "xlsx")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	defer os.Remove(xlsxPath) // le classeur est lu en mémoire ci-dessous

	sheetName := strings.TrimSpace(r.FormValue("sheet"))
	filterCol := strings.TrimSpace(r.FormValue("filterCol"))

	// Format de sortie : "pdf" (défaut) ou "docx".
	format := strings.TrimSpace(r.FormValue("format"))
	if format == "" {
		format = "pdf"
	}
	if format != "pdf" && format != "docx" {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("format inconnu : %q (attendu pdf ou docx)", format))
		return
	}
	if format == "pdf" && h.gotenbergURL == "" {
		writeJSONError(w, http.StatusServiceUnavailable, fmt.Errorf("conversion PDF indisponible : gotenbergURL non configuré"))
		return
	}

	book, err := ouvrirClasseur(xlsxPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("lecture du classeur : %w", err))
		return
	}
	f, ok := book.sheets[sheetName]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("feuille introuvable : %q", sheetName))
		return
	}

	total := compterEleves(f, filterCol)
	if total == 0 {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("aucun élève à générer (filtre trop restrictif ?)"))
		return
	}

	// Le template doit survivre à la requête : la goroutine le supprimera.
	tmplPath, err := saveUpload(r, "template")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	j := h.jobs.create(total, format)
	go h.runJob(j.ID, f, tmplPath, filterCol, format)

	writeJSON(w, map[string]string{"id": j.ID})
}

// handleGenerateStatus renvoie l'état courant d'un job (state, done, total).
func (h *handlers) handleGenerateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	j, ok := h.jobs.snapshot(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, fmt.Errorf("job introuvable : %q", id))
		return
	}
	writeJSON(w, j)
}

// handleGenerateDownload sert le zip d'un job terminé.
func (h *handlers) handleGenerateDownload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	j, ok := h.jobs.snapshot(id)
	if !ok {
		writeJSONError(w, http.StatusNotFound, fmt.Errorf("job introuvable : %q", id))
		return
	}
	if j.State != jobDone || j.zipPath == "" {
		writeJSONError(w, http.StatusConflict, fmt.Errorf("génération non terminée (état : %s)", j.State))
		return
	}

	file, err := os.Open(j.zipPath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="bulletins-%s.zip"`, j.Format))
	if _, err := io.Copy(w, file); err != nil {
		log.Println("erreur envoi zip :", err)
	}
}

// compterEleves compte les lignes retenues par le filtre (pour connaître le
// total avant de lancer la génération).
func compterEleves(f *feuille, filterCol string) int {
	const premiereLigne = 4
	n := 0
	for row := premiereLigne; f.cell("A", row) != ""; row++ {
		if filtrePasse(f, filterCol, row) {
			n++
		}
	}
	return n
}

// runJob génère les bulletins (docx, puis PDF si demandé), les zippe dans un
// fichier temporaire et met à jour la progression du job. Tourne en goroutine.
func (h *handlers) runJob(id string, f *feuille, tmplPath, filterCol, format string) {
	defer os.Remove(tmplPath)

	// outDir : fichiers finaux à zipper. En PDF, les .docx intermédiaires sont
	// écrits dans docxDir puis convertis ; en docx, le .docx final EST dans outDir.
	outDir, err := os.MkdirTemp("", "bulletins-"+id+"-*")
	if err != nil {
		h.jobs.fail(id, err.Error())
		return
	}
	defer os.RemoveAll(outDir)

	docxDir := outDir
	if format == "pdf" {
		docxDir, err = os.MkdirTemp("", "bulletins-docx-"+id+"-*")
		if err != nil {
			h.jobs.fail(id, err.Error())
			return
		}
		defer os.RemoveAll(docxDir)
	}

	const premiereLigne = 4
	for row := premiereLigne; f.cell("A", row) != ""; row++ {
		if !filtrePasse(f, filterCol, row) {
			continue // élève exclu par le filtre
		}

		nom := strings.TrimSpace(f.cell("A", row))
		prenom := strings.TrimSpace(f.cell("B", row))
		base := sanitize(nom + "." + prenom)

		docxPath := filepath.Join(docxDir, base+".docx")
		if err := genererBulletin(tmplPath, docxPath, f, row); err != nil {
			h.jobs.fail(id, fmt.Sprintf("génération %s %s : %v", nom, prenom, err))
			return
		}

		if format == "pdf" {
			pdfPath := filepath.Join(outDir, base+".pdf")
			if err := convertirDocxEnPDF(context.Background(), h.gotenbergURL, docxPath, pdfPath); err != nil {
				h.jobs.fail(id, fmt.Sprintf("conversion PDF %s %s : %v", nom, prenom, err))
				return
			}
		}
		h.jobs.incProgress(id)
	}

	// Zip vers un fichier temporaire, conservé jusqu'au download (ou au TTL).
	zipFile, err := os.CreateTemp("", "bulletins-"+id+"-*.zip")
	if err != nil {
		h.jobs.fail(id, err.Error())
		return
	}
	zipPath := zipFile.Name()
	if err := zipDir(zipFile, outDir); err != nil {
		zipFile.Close()
		os.Remove(zipPath)
		h.jobs.fail(id, err.Error())
		return
	}
	if err := zipFile.Close(); err != nil {
		os.Remove(zipPath)
		h.jobs.fail(id, err.Error())
		return
	}

	h.jobs.finish(id, zipPath)
}

// zipDir écrit dans w une archive zip contenant tous les fichiers de dir.
func zipDir(w io.Writer, dir string) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := addFileToZip(zw, filepath.Join(dir, entry.Name()), entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zw *zip.Writer, path, name string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("erreur encodage JSON :", err)
	}
}
