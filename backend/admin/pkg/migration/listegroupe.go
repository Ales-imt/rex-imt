package migration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// parseListeGroupe analyse le fichier listegroupe_html?MODE=10.
//
// Format observé (encodé Windows-1252, délimiteur tabulation) :
//
//	Promotion :     1A INFRES       Groupe :        INFRES  Matière :       ...
//	Num étudiant    Nom     Prénom  note
//	19475   AHMED ALI       Adoita
//
// Règles :
//   - Décoder en Windows-1252.
//   - Détecter le délimiteur sur les octets bruts (tabulation en priorité, sinon ';').
//   - Ignorer toutes les lignes dont le premier champ n'est pas numérique.
//   - Retourner uniquement le numéro d'étudiant (EV) ; la colonne note est ignorée.
//   - Retourner aussi les libellés Groupe et Promotion du préambule pour validation.
func parseListeGroupe(raw []byte) (evs []string, groupeLabel, promoLabel string) {
	// Détecter délimiteur sur les octets bruts.
	delim := detectDelimiter(raw)

	decoder := charmap.Windows1252.NewDecoder()
	decoded, _, _ := transform.Bytes(decoder, raw)

	for _, line := range strings.Split(string(decoded), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Extraction du préambule (première ligne non vide).
		if promoLabel == "" && strings.Contains(line, "Promotion") {
			promoLabel = extractLabel(line, "Promotion")
			groupeLabel = extractLabel(line, "Groupe")
			continue
		}
		fields := strings.Split(line, string(delim))
		if len(fields) == 0 {
			continue
		}
		first := strings.TrimSpace(fields[0])
		if !isNumeric(first) {
			continue
		}
		evs = append(evs, first)
	}
	return
}

// detectDelimiter inspecte les octets bruts pour choisir '\t' ou ';'.
func detectDelimiter(raw []byte) byte {
	if bytes.ContainsRune(raw, '\t') {
		return '\t'
	}
	return ';'
}

// extractLabel extrait la valeur après le mot-clé dans une ligne de préambule.
// Ex : "Promotion :     1A INFRES       Groupe :  INFRES" → "1A INFRES" pour "Promotion".
func extractLabel(line, keyword string) string {
	idx := strings.Index(line, keyword)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(keyword):]
	// Sauter le séparateur éventuel (':' et espaces).
	rest = strings.TrimLeft(rest, " :\t")
	// Arrêter au prochain mot-clé (Groupe, Matière, Contrôle…).
	for _, stop := range []string{"Groupe", "Matière", "Contrôle", "Coefficient", "\t"} {
		if i := strings.Index(rest, stop); i >= 0 {
			rest = rest[:i]
		}
	}
	return strings.TrimSpace(rest)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// SyncElevesGroupe synchronise l'appartenance élève↔groupe pour tous les groupes
// webdfd connus de l'année courante.
//
// Pour chaque groupe :
//  1. Télécharge listegroupe_html&MODE=10.
//  2. Parse et valide les libellés du préambule.
//  3. Résout chaque EV via migration.user_map ; ignore les EV inconnus (log + skip).
//  4. Upsert eleve_groupe.
//  5. Supprime les entrées eleve_groupe qui ne sont plus dans la liste.
//  6. Met à jour groupe.taille.
func SyncElevesGroupe(ctx context.Context, listeGroupeBaseURL string, db *pgxpool.Pool, ac anneeCourante) error {
	q := New(db)
	annee := ac.Annee

	groupes, err := q.ListGroupeWebdfdIDs(ctx, annee)
	if err != nil {
		return err
	}

	for _, gr := range groupes {
		if err := syncGroupe(ctx, listeGroupeBaseURL, gr.InternalID, gr.ExternalID, annee, q); err != nil {
			log.Printf("eleve_groupe: groupe GRCLE=%s: %v — continué", gr.ExternalID, err)
		}
	}
	return nil
}

func syncGroupe(ctx context.Context, baseURL string, internalID int64, grcle string, annee int32, q *Queries) error {
	url := fmt.Sprintf("%s?TYPE=listegroupe_html&GRCLEUNIK=%s&MODE=10", baseURL, grcle)
	resp, err := webdfdGet(url)
	if err != nil {
		return fmt.Errorf("webdfd: listegroupe GRCLE=%s inaccessible: %w", grcle, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	evs, groupeLabel, _ := parseListeGroupe(raw)

	// Garde-fou : vérifier la cohérence du libellé si disponible.
	if groupeLabel != "" {
		expectedLabel, err := q.GetGroupeLabel(ctx, internalID)
		if err == nil && expectedLabel != "" && !strings.EqualFold(groupeLabel, expectedLabel) {
			log.Printf("eleve_groupe: GRCLE=%s: libellé flux %q ≠ DB %q (GRCLE périmé ?)", grcle, groupeLabel, expectedLabel)
		}
	}

	idGroupe := int32(internalID)
	var seen []int32
	var skipped int
	for _, ev := range evs {
		userID, err := q.GetUserBySource(ctx, GetUserBySourceParams{Source: "webdfd", ExternalID: ev})
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("eleve_groupe: EV=%s introuvable dans user_map, ignoré (rechargez eleves_txt)", ev)
			skipped++
			continue
		}
		if err != nil {
			return fmt.Errorf("user_map: lookup EV=%s: %w", ev, err)
		}
		if err = q.UpsertEleveGroupe(ctx, UpsertEleveGroupeParams{NumEtudiant: userID, IDGroupe: idGroupe}); err != nil {
			return err
		}
		seen = append(seen, userID)
	}

	// Réconciliation : retirer les membres absents de la liste courante.
	if err = q.DeleteEleveGroupeAbsents(ctx, DeleteEleveGroupeAbsentsParams{
		IDGroupe: idGroupe,
		Column2:  seen,
	}); err != nil {
		return err
	}

	taille := len(seen)
	if err = q.UpdateGroupeTaille(ctx, UpdateGroupeTailleParams{Taille: int32(taille), ID: internalID}); err != nil {
		return err
	}

	log.Printf("eleve_groupe: GRCLE=%s: %d élèves (%d ignorés)", grcle, taille, skipped)
	return nil
}
