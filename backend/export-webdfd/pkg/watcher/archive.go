package watcher

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// VerifyZip verifie que le fichier est bien une archive zip lisible et que
// toutes ses entrees ont un CRC correct. C'est l'equivalent de `unzip -t` :
// une archive tronquee (copie encore en cours, transfert interrompu) est
// rejetee ici plutot que de produire un export partiel.
func VerifyZip(name string) error {
	r, err := zip.OpenReader(name)
	if err != nil {
		return fmt.Errorf("archive illisible : %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("%s : %w", f.Name, err)
		}
		// archive/zip controle le CRC32 en fin de lecture.
		_, err = io.Copy(io.Discard, rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("%s : %w", f.Name, err)
		}
	}
	return nil
}

// ZipRoot renvoie le repertoire de premier niveau commun a toutes les entrees
// de l'archive. L'export attend une base HFSQL complete dans un repertoire
// unique (ex. 20250707.bd) : une archive plate ou a plusieurs racines est une
// erreur de depot, pas quelque chose a deviner.
func ZipRoot(name string) (string, error) {
	r, err := zip.OpenReader(name)
	if err != nil {
		return "", fmt.Errorf("archive illisible : %w", err)
	}
	defer r.Close()

	root := ""
	for _, f := range r.File {
		clean := path.Clean(f.Name)
		first, rest, _ := strings.Cut(clean, "/")
		if first == "" || first == "." || rest == "" && !f.FileInfo().IsDir() {
			return "", fmt.Errorf("l'archive contient %q hors d'un repertoire racine", f.Name)
		}
		if root == "" {
			root = first
		} else if first != root {
			return "", fmt.Errorf("l'archive contient plusieurs repertoires racine (%q et %q)", root, first)
		}
	}
	if root == "" {
		return "", errors.New("archive vide")
	}
	return root, nil
}

// Unzip extrait l'archive dans dest. Les chemins sortant de dest (« zip slip »)
// et les entrees qui ne sont ni fichier ni repertoire sont rejetes.
func Unzip(name, dest string) error {
	r, err := zip.OpenReader(name)
	if err != nil {
		return fmt.Errorf("archive illisible : %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		clean := path.Clean(f.Name)
		if !filepath.IsLocal(clean) {
			return fmt.Errorf("entree hors du repertoire cible : %q", f.Name)
		}
		target := filepath.Join(dest, filepath.FromSlash(clean))

		info := f.FileInfo()
		switch {
		case info.IsDir():
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if err := extractFile(f, target); err != nil {
				return fmt.Errorf("%s : %w", f.Name, err)
			}
		default:
			return fmt.Errorf("entree de type non supporte : %q", f.Name)
		}
	}
	return nil
}

func extractFile(f *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	mode := f.FileInfo().Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
