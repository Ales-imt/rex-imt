// Package hfsqlexport décrit la convention de nommage des exports HFSQL, seul
// point de contact entre le service qui les produit (export-webdfd) et celui
// qui les consomme (sync).
//
// Le contrat est un répertoire partagé, pas un appel réseau :
//
//	<data>/export-AAAAMMJJ-HHMMSS/json/*.json   les tables exportées
//	<data>/export-AAAAMMJJ-HHMMSS/.termine      écrit EN DERNIER
//
// Le marqueur est indispensable : un export dure plusieurs minutes, et le
// répertoire existe dès la première seconde. Sans lui, le consommateur lirait
// tôt ou tard un export à moitié écrit — un planning amputé, donc des séances
// annulées à tort.
//
// Vivre dans common plutôt que d'être recopié de part et d'autre est
// délibéré : deux définitions divergentes de cette convention ne provoqueraient
// aucune erreur de compilation, juste une synchronisation qui ne se ferait
// plus.
package hfsqlexport

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// Prefixe ouvre le nom des répertoires d'export. Le suffixe est horodaté au
	// format Horodatage, si bien que l'ordre lexicographique des noms est
	// l'ordre chronologique : trier suffit, il n'y a jamais à interroger les
	// dates du système de fichiers.
	Prefixe = "export-"

	// Horodatage est le format du suffixe (time.Layout).
	Horodatage = "20060102-150405"

	// SousDossierJSON contient les tables exportées.
	SousDossierJSON = "json"

	// Marqueur signale un export complet. Écrit en dernier, une fois tous les
	// fichiers fermés.
	Marqueur = ".termine"
)

// NomExport rend le nom de répertoire d'un export daté de t.
func NomExport(t time.Time) string { return Prefixe + t.Format(Horodatage) }

// Lister rend les répertoires d'export présents sous data, du plus ancien au
// plus récent, marqueur présent ou non.
func Lister(data string) ([]string, error) {
	entries, err := os.ReadDir(data)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", data, err)
	}
	var noms []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), Prefixe) {
			noms = append(noms, e.Name())
		}
	}
	sort.Strings(noms)
	return noms, nil
}

// Complet indique si l'export porte son marqueur de fin.
func Complet(data, nom string) bool {
	_, err := os.Stat(filepath.Join(data, nom, Marqueur))
	return err == nil
}

// Marquer pose le marqueur de fin sur un export. À n'appeler qu'une fois
// l'export réussi et tous ses fichiers fermés.
func Marquer(data, nom string) error {
	chemin := filepath.Join(data, nom, Marqueur)
	if err := os.WriteFile(chemin, []byte(time.Now().Format(time.RFC3339)+"\n"), 0o644); err != nil {
		return fmt.Errorf("écriture de %s : %w", chemin, err)
	}
	return nil
}

// Dernier rend le nom et le répertoire json de l'export complet le plus récent
// sous data. nom est vide si aucun export complet n'y figure — ce n'est pas une
// erreur : au premier démarrage, rien n'a encore été déposé.
//
// Un export interrompu (sans marqueur) est ignoré et l'on remonte au précédent,
// plutôt que de servir un instantané tronqué.
func Dernier(data string) (nom, jsonDir string, err error) {
	noms, err := Lister(data)
	if err != nil {
		return "", "", err
	}
	for i := len(noms) - 1; i >= 0; i-- {
		if !Complet(data, noms[i]) {
			continue
		}
		dir := filepath.Join(data, noms[i], SousDossierJSON)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return noms[i], dir, nil
		}
	}
	return "", "", nil
}

// Purger ne conserve que les keep exports les plus récents sous data.
// keep <= 0 désactive la purge.
//
// Chaque dépôt laisse un export complet en plus de la base dézippée : sans
// purge, le disque se remplit au rythme des sauvegardes.
func Purger(data string, keep int) error {
	if keep <= 0 {
		return nil
	}
	noms, err := Lister(data)
	if err != nil {
		return err
	}
	if len(noms) <= keep {
		return nil
	}
	for _, nom := range noms[:len(noms)-keep] {
		dir := filepath.Join(data, nom)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("suppression de %s : %w", dir, err)
		}
	}
	return nil
}
