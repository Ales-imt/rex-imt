package presence

// Export semestriel : assemblage d'un document couvrant toutes les feuilles de
// présence d'une promotion sur une période.
//
// Coût en base, quel que soit le nombre de séances : une requête d'en-tête, une
// de séances, deux de présence, deux pour l'annexe registre. Le regroupement
// par séance se fait ici, en mémoire.

import (
	presencegen "back-rex-common/pkg/presencedata/gen"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// exportParams : filtres et options d'un export, tels que reçus en query string.
type exportParams struct {
	PeriodeID  int64
	MatiereIDs []int64 // vide = toutes les matières de la période
	From, To   pgtype.Timestamptz
	Options    pdfOptions
}

// buildSemestreDoc lit la base et compose le document. Les séances non
// clôturées sont écartées du corps mais restituées nommément dans doc.Exclues :
// un export de cent séances ne doit pas échouer parce que l'une est restée
// ouverte, mais son absence ne doit pas non plus passer inaperçue.
func buildSemestreDoc(ctx context.Context, db DBTX, p exportParams) (pdfDoc, GetPeriodeHeaderRow, error) {
	q := New(db)
	qd := presencegen.New(db)

	header, err := q.GetPeriodeHeader(ctx, p.PeriodeID)
	if err != nil {
		return pdfDoc{}, header, fmt.Errorf("en-tête de période : %w", err)
	}

	seances, err := q.ListSeancesByPeriode(ctx, ListSeancesByPeriodeParams{
		PeriodeID:  p.PeriodeID,
		MatiereIds: p.MatiereIDs,
		FromTs:     p.From,
		ToTs:       p.To,
	})
	if err != nil {
		return pdfDoc{}, header, fmt.Errorf("séances de la période : %w", err)
	}

	doc := pdfDoc{
		Promo:       header.PromoName,
		Periode:     header.PeriodeName,
		Annee:       header.Annee,
		GeneratedAt: time.Now().UTC(),
		Options:     p.Options,
	}

	retenues := make([]ListSeancesByPeriodeRow, 0, len(seances))
	for _, s := range seances {
		if s.ClosedAt.Valid {
			retenues = append(retenues, s)
			continue
		}
		doc.Exclues = append(doc.Exclues, libelleSeanceExclue(s))
	}
	if len(retenues) == 0 {
		return doc, header, nil
	}

	ids := make([]int64, 0, len(retenues))
	for _, s := range retenues {
		ids = append(ids, s.ID)
	}

	presences, err := qd.ListPresenceBySeances(ctx, ids)
	if err != nil {
		return pdfDoc{}, header, fmt.Errorf("présences : %w", err)
	}
	horsGroupe, err := qd.ListPresenceHorsGroupeBySeances(ctx, ids)
	if err != nil {
		return pdfDoc{}, header, fmt.Errorf("présences hors groupe : %w", err)
	}

	parSeance := make(map[int64][]presenceLigne, len(retenues))
	for _, r := range presences {
		parSeance[r.SeanceID] = append(parSeance[r.SeanceID], presenceLigne{
			UserID: r.UserID, Name: r.Name, Surname: r.Surname,
			Statut: r.Statut, PointeAt: r.PointeAt, Justifie: r.Justifie,
		})
	}
	hgParSeance := make(map[int64][]presenceLigne)
	for _, r := range horsGroupe {
		hgParSeance[r.SeanceID] = append(hgParSeance[r.SeanceID], presenceLigne{
			UserID: r.UserID, Name: r.Name, Surname: r.Surname,
			Statut: r.Statut, PointeAt: r.PointeAt, Justifie: r.Justifie,
		})
	}

	doc.Seances = make([]pdfData, 0, len(retenues))
	for _, s := range retenues {
		doc.Seances = append(doc.Seances, buildPdfData(
			seanceInfoFromPeriode(s),
			parSeance[s.ID],
			hgParSeance[s.ID],
			doc.GeneratedAt,
		))
	}

	if p.Options.Ledger {
		doc.Ledger, err = buildLedgerRows(ctx, q, ids)
		if err != nil {
			return pdfDoc{}, header, err
		}
	}

	return doc, header, nil
}

// buildLedgerRows rapproche les bornes du registre de chaque séance de la
// première ancre TSA de rang supérieur ou égal : celle-ci scelle, par chaînage,
// tous les maillons antérieurs.
func buildLedgerRows(ctx context.Context, q *Queries, ids []int64) ([]pdfLedgerRow, error) {
	entrees, err := q.ListLedgerBySeances(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("registre : %w", err)
	}
	ancres, err := q.ListAnchorSeqs(ctx)
	if err != nil {
		return nil, fmt.Errorf("ancres : %w", err)
	}

	rows := make([]pdfLedgerRow, 0, len(entrees))
	for _, e := range entrees {
		row := pdfLedgerRow{
			SeanceID: e.SeanceID, SeqMin: e.SeqMin, SeqMax: e.SeqMax,
			NbMaillons: e.NbMaillons, Hash: e.DernierHash,
		}
		// ListAnchorSeqs est trié : recherche dichotomique de la première ancre
		// couvrant le dernier maillon de la séance.
		i := sort.Search(len(ancres), func(i int) bool {
			return ancres[i].LedgerSeq >= e.SeqMax
		})
		if i < len(ancres) {
			row.AncreSeq = ancres[i].LedgerSeq
			if ancres[i].CreatedAt.Valid {
				row.AncreAt = ancres[i].CreatedAt.Time.In(parisLoc).Format("02/01/2006")
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// libelleSeanceExclue décrit une séance écartée de façon à la retrouver dans le
// planning : matière, date et horaire, pas seulement un identifiant.
func libelleSeanceExclue(s ListSeancesByPeriodeRow) string {
	parts := []string{s.MatiereName}
	if s.StartsAt.Valid {
		parts = append(parts, formatDateFR(s.StartsAt.Time))
		horaire := formatHeure(s.StartsAt.Time)
		if s.EndsAt.Valid {
			horaire += " – " + formatHeure(s.EndsAt.Time)
		}
		parts = append(parts, horaire)
	}
	if s.Salle != "" {
		parts = append(parts, s.Salle)
	}
	return strings.Join(parts, " · ")
}

// ─── Nom de fichier ───────────────────────────────────────────────────────────

// exportFilename : « presence-ing1-s2-2025-2026.pdf ». Un nom lisible est
// attendu par l'utilisateur dans son dossier de téléchargements ; un
// « presence-periode-42.pdf » ne lui apprend rien.
func exportFilename(h GetPeriodeHeaderRow) string {
	parts := []string{"presence"}
	if s := slug(h.PromoName); s != "" {
		parts = append(parts, s)
	}
	if s := slug(h.PeriodeName); s != "" {
		parts = append(parts, s)
	}
	if h.Annee > 0 {
		parts = append(parts, fmt.Sprintf("%d-%d", h.Annee, h.Annee+1))
	} else {
		parts = append(parts, strconv.FormatInt(h.ID, 10))
	}
	return strings.Join(parts, "-") + ".pdf"
}

// slug déplie les accents puis réduit à [a-z0-9-]. Le nom de fichier traverse
// un en-tête HTTP et des systèmes de fichiers variés : on n'y laisse pas
// d'octets non ASCII.
func slug(s string) string {
	sansAccents, _, err := transform.String(
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
		s,
	)
	if err != nil {
		sansAccents = s
	}

	var b strings.Builder
	tiret := false
	for _, r := range strings.ToLower(sansAccents) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			tiret = false
		default:
			if !tiret && b.Len() > 0 {
				b.WriteByte('-')
				tiret = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
