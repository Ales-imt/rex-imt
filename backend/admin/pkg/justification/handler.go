package justification

// Handlers des justifications d'absence (« excuses »).
//
// Ces routes sont montées dans back-rex-admin UNIQUEMENT : la saisie est un
// acte de gestionnaire d'année. Le service étudiant n'expose que la lecture du
// marquage sur ses écrans prof (colonne justifie de ListPresence) ; un prof qui
// constate une erreur passe par le gestionnaire, hors logiciel — comme le motif.
//
// Aucun de ces handlers n'écrit dans `pointage` ni dans `presence_ledger`.

import (
	"back-rex-common/pkg/auth"
	"back-rex-common/pkg/services"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// uniqueViolation : SQLSTATE d'une violation d'unicité. Deux cas ici, tous deux
// des courses entre gestionnaires : double révocation (PK de
// justification_revocation) et double correction de la même version
// (uq_justification_replaces).
const uniqueViolation = "23505"

// ─── Types de réponse ─────────────────────────────────────────────────────────

type seanceItem struct {
	ID       int64  `json:"id"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
	Matiere  string `json:"matiere"`
	Salle    string `json:"salle"`
	Statut   string `json:"statut"`
	// DansPlage : la séance tombe dans la plage demandée. DejaCouverte : elle
	// est actuellement couverte par la justification en cours de modification.
	// Le dialogue en déduit les entrantes (dans la plage, pas encore couvertes)
	// et les sortantes (couvertes, plus dans la plage) — la seconde catégorie
	// n'existe qu'en modification.
	DansPlage    bool `json:"dans_plage"`
	DejaCouverte bool `json:"deja_couverte"`
}

type chevauchementItem struct {
	ID       int64  `json:"id"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

type previewResponse struct {
	Seances        []seanceItem        `json:"seances"`
	Chevauchements []chevauchementItem `json:"chevauchements"`
}

type justificationItem struct {
	ID               int64   `json:"id"`
	UserID           int32   `json:"user_id"`
	Name             string  `json:"name"`
	Surname          string  `json:"surname"`
	StartsAt         string  `json:"starts_at"`
	EndsAt           string  `json:"ends_at"`
	NbSeances        int64   `json:"nb_seances"`
	Statut           string  `json:"statut"` // ACTIVE | ANNULEE | REMPLACEE
	CreatedAt        string  `json:"created_at"`
	CreatedBy        int32   `json:"created_by"`
	CreatedByName    string  `json:"created_by_name"`
	CreatedBySurname string  `json:"created_by_surname"`
	RevokedAt        *string `json:"revoked_at"`
	ReplacesID       *int64  `json:"replaces_id"`
	ReplacedByID     *int64  `json:"replaced_by_id"`
}

type saisieRequest struct {
	UserID   int32  `json:"user_id"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
	// SeanceIds, optionnel : restreint la couverture aux séances retenues par
	// le gestionnaire dans l'aperçu. Absent = toutes les séances recouvertes.
	SeanceIds *[]int64 `json:"seance_ids"`
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func fmtTs(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func fmtTsPtr(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

func int8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func parseID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	return id, err == nil && id > 0
}

// queryInt8 lit un filtre entier optionnel de query string. Absent ou illisible
// vaut « pas de filtre » (NULL), jamais zéro : un zéro serait un identifiant.
func queryInt8(r *http.Request, nom string) pgtype.Int8 {
	v := r.URL.Query().Get(nom)
	if v == "" {
		return pgtype.Int8{}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: n, Valid: true}
}

func queryInt4(r *http.Request, nom string) pgtype.Int4 {
	v := queryInt8(r, nom)
	if !v.Valid {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(v.Int64), Valid: true}
}

func toItem(row ListJustificationsRow) justificationItem {
	return justificationItem{
		ID:               row.ID,
		UserID:           row.UserID,
		Name:             row.Name,
		Surname:          row.Surname,
		StartsAt:         fmtTs(row.Debut),
		EndsAt:           fmtTs(row.Fin),
		NbSeances:        row.NbSeances,
		Statut:           row.Statut,
		CreatedAt:        fmtTs(row.CreatedAt),
		CreatedBy:        row.CreatedBy,
		CreatedByName:    row.CreatedByName,
		CreatedBySurname: row.CreatedBySurname,
		RevokedAt:        fmtTsPtr(row.RevokedAt),
		ReplacesID:       int8Ptr(row.ReplacesID),
		ReplacedByID:     int8Ptr(row.ReplacedByID),
	}
}

func getToItem(row GetJustificationRow) justificationItem {
	return toItem(ListJustificationsRow(row))
}

// ─── Aperçu ──────────────────────────────────────────────────────────────────

// PreviewHandler renvoie les séances recouvertes par une plage, chacune avec le
// statut de pointage actuel de l'étudiant, ainsi que les justifications actives
// qui chevauchent déjà cette plage. C'est le cœur du dialogue de saisie : la
// règle de chevauchement y devient inspectable plutôt que d'être devinée.
// GET /justifications/preview?user_id=&starts_at=&ends_at=&exclude_id=
func PreviewHandler(w http.ResponseWriter, r *http.Request) {
	userID := queryInt4(r, "user_id")
	if !userID.Valid {
		services.InvalidRequestError(w, r, "user_id requis", services.NO_INFORMATION, nil)
		return
	}
	periode, err := makePeriode(r.URL.Query().Get("starts_at"), r.URL.Query().Get("ends_at"))
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	excludeID := queryInt8(r, "exclude_id")

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	couvertes, err := q.ListSeancesCouvertes(ctx, ListSeancesCouvertesParams{
		UserID:  userID.Int32,
		Periode: periode,
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	// En modification, la couverture actuelle sert à marquer les entrantes et
	// les sortantes. Les sortantes sont hors plage : elles ne figurent pas dans
	// `couvertes` et doivent être ajoutées à la liste pour être visibles.
	actuelles := map[int64]ListJustificationSeancesRow{}
	if excludeID.Valid {
		rows, err := q.ListJustificationSeances(ctx, excludeID.Int64)
		if err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
		for _, s := range rows {
			actuelles[s.ID] = s
		}
	}

	seances := make([]seanceItem, 0, len(couvertes)+len(actuelles))
	dansPlage := make(map[int64]bool, len(couvertes))
	for _, s := range couvertes {
		dansPlage[s.ID] = true
		_, deja := actuelles[s.ID]
		seances = append(seances, seanceItem{
			ID:           s.ID,
			StartsAt:     fmtTs(s.StartsAt),
			EndsAt:       fmtTs(s.EndsAt),
			Matiere:      s.MatiereName,
			Salle:        s.Salle,
			Statut:       s.Statut,
			DansPlage:    true,
			DejaCouverte: deja,
		})
	}
	for id, s := range actuelles {
		if dansPlage[id] {
			continue
		}
		seances = append(seances, seanceItem{
			ID:           s.ID,
			StartsAt:     fmtTs(s.StartsAt),
			EndsAt:       fmtTs(s.EndsAt),
			Matiere:      s.MatiereName,
			Salle:        s.Salle,
			Statut:       s.Statut,
			DansPlage:    false,
			DejaCouverte: true,
		})
	}

	chevauchements, err := q.ListJustificationsChevauchantes(ctx, ListJustificationsChevauchantesParams{
		UserID:    userID.Int32,
		Periode:   periode,
		ExcludeID: excludeID,
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	conflits := make([]chevauchementItem, 0, len(chevauchements))
	for _, c := range chevauchements {
		conflits = append(conflits, chevauchementItem{
			ID: c.ID, StartsAt: fmtTs(c.Debut), EndsAt: fmtTs(c.Fin),
		})
	}

	render.JSON(w, r, previewResponse{Seances: seances, Chevauchements: conflits})
}

// ─── Écriture ────────────────────────────────────────────────────────────────

// couverture résout la liste des séances à rattacher : l'intersection entre la
// sélection du client et les séances réellement recouvertes. L'intersection
// n'est pas une précaution de style — sans elle, un client pourrait rattacher
// n'importe quelle séance, y compris celle d'une autre promotion.
func couverture(ctx context.Context, q *Queries, userID int32, periode pgtype.Range[pgtype.Timestamptz], choix *[]int64) ([]int64, error) {
	couvertes, err := q.ListSeancesCouvertes(ctx, ListSeancesCouvertesParams{UserID: userID, Periode: periode})
	if err != nil {
		return nil, err
	}
	if choix == nil {
		ids := make([]int64, 0, len(couvertes))
		for _, s := range couvertes {
			ids = append(ids, s.ID)
		}
		return ids, nil
	}

	retenu := make(map[int64]bool, len(*choix))
	for _, id := range *choix {
		retenu[id] = true
	}
	ids := make([]int64, 0, len(couvertes))
	for _, s := range couvertes {
		if retenu[s.ID] {
			ids = append(ids, s.ID)
		}
	}
	return ids, nil
}

// lireSaisie décode et valide le corps commun à la création et à la modification.
func lireSaisie(w http.ResponseWriter, r *http.Request) (saisieRequest, pgtype.Range[pgtype.Timestamptz], bool) {
	var req saisieRequest
	var periode pgtype.Range[pgtype.Timestamptz]

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		services.InvalidRequestError(w, r, "corps invalide", services.NO_INFORMATION, nil)
		return req, periode, false
	}
	if req.UserID <= 0 {
		services.InvalidRequestError(w, r, "user_id requis", services.NO_INFORMATION, nil)
		return req, periode, false
	}
	periode, err := makePeriode(req.StartsAt, req.EndsAt)
	if err != nil {
		services.InvalidRequestError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return req, periode, false
	}
	return req, periode, true
}

// CreateHandler enregistre une nouvelle justification et sa couverture.
// POST /justifications
func CreateHandler(w http.ResponseWriter, r *http.Request) {
	req, periode, ok := lireSaisie(w, r)
	if !ok {
		return
	}
	auteur := auth.GetSecurityUserId(r.Context())
	if auteur == nil {
		services.AuthenticationError(w, r, "utilisateur inconnu", services.NO_INFORMATION, nil)
		return
	}

	db := services.GetPgCtx(r.Context()).Db
	ctx := context.Background()
	q := New(db)

	seanceIDs, err := couverture(ctx, q, req.UserID, periode, req.SeanceIds)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	defer tx.Rollback(ctx)
	qtx := q.WithTx(tx)

	cree, err := qtx.CreateJustification(ctx, CreateJustificationParams{
		UserID:    req.UserID,
		Periode:   periode,
		CreatedBy: int32(*auteur),
	})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if len(seanceIDs) > 0 {
		if err := qtx.InsertJustificationSeances(ctx, InsertJustificationSeancesParams{
			JustificationID: cree.ID,
			SeanceIds:       seanceIDs,
		}); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	rendreJustification(w, r, q, cree.ID, http.StatusCreated)
}

// UpdateHandler « modifie » une justification. Les tables étant append-only, ce
// n'est jamais un UPDATE : dans une seule transaction, l'ancienne est révoquée
// et une nouvelle version est insérée avec replaces_id vers elle. On ne cherche
// pas à n'ajuster que le delta de séances — la simplicité de raisonnement prime,
// le volume est négligeable.
// PUT /justifications/{id}
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	ancienID, ok := parseID(r)
	if !ok {
		services.InvalidRequestError(w, r, "id invalide", services.NO_INFORMATION, nil)
		return
	}
	req, periode, ok := lireSaisie(w, r)
	if !ok {
		return
	}
	auteur := auth.GetSecurityUserId(r.Context())
	if auteur == nil {
		services.AuthenticationError(w, r, "utilisateur inconnu", services.NO_INFORMATION, nil)
		return
	}

	db := services.GetPgCtx(r.Context()).Db
	ctx := context.Background()
	q := New(db)

	ancien, err := q.GetJustification(ctx, ancienID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "justification introuvable", http.StatusNotFound)
		return
	}
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	if ancien.Statut != "ACTIVE" {
		// Corriger une version déjà annulée ou déjà remplacée produirait une
		// seconde version active de la même excuse.
		services.ConflictError(w, r, "cette excuse n'est plus active : reprendre la version en vigueur",
			services.NO_INFORMATION, nil)
		return
	}

	seanceIDs, err := couverture(ctx, q, req.UserID, periode, req.SeanceIds)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	defer tx.Rollback(ctx)
	qtx := q.WithTx(tx)

	if err := qtx.RevokeJustification(ctx, RevokeJustificationParams{
		JustificationID: ancienID,
		RevokedBy:       int32(*auteur),
	}); err != nil {
		rendreErreurEcriture(w, r, err)
		return
	}
	cree, err := qtx.CreateJustification(ctx, CreateJustificationParams{
		UserID:     req.UserID,
		Periode:    periode,
		ReplacesID: pgtype.Int8{Int64: ancienID, Valid: true},
		CreatedBy:  int32(*auteur),
	})
	if err != nil {
		rendreErreurEcriture(w, r, err)
		return
	}
	if len(seanceIDs) > 0 {
		if err := qtx.InsertJustificationSeances(ctx, InsertJustificationSeancesParams{
			JustificationID: cree.ID,
			SeanceIds:       seanceIDs,
		}); err != nil {
			services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		rendreErreurEcriture(w, r, err)
		return
	}

	rendreJustification(w, r, q, cree.ID, http.StatusOK)
}

// DeleteHandler annule une excuse : une ligne dans justification_revocation, et
// rien d'autre. Les séances concernées redeviennent « Absent » à la lecture
// suivante ; aucune ligne n'est supprimée.
// DELETE /justifications/{id}
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		services.InvalidRequestError(w, r, "id invalide", services.NO_INFORMATION, nil)
		return
	}
	auteur := auth.GetSecurityUserId(r.Context())
	if auteur == nil {
		services.AuthenticationError(w, r, "utilisateur inconnu", services.NO_INFORMATION, nil)
		return
	}

	q := New(services.GetPgCtx(r.Context()).Db)
	ctx := context.Background()

	if _, err := q.GetJustification(ctx, id); errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "justification introuvable", http.StatusNotFound)
		return
	} else if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	if err := q.RevokeJustification(ctx, RevokeJustificationParams{
		JustificationID: id,
		RevokedBy:       int32(*auteur),
	}); err != nil {
		rendreErreurEcriture(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// rendreErreurEcriture traduit une violation d'unicité en 409 : elle signale
// systématiquement une course entre deux gestionnaires (double annulation, ou
// deux corrections de la même version), jamais une erreur serveur.
func rendreErreurEcriture(w http.ResponseWriter, r *http.Request, err error) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		services.ConflictError(w, r, "cette excuse vient d'être modifiée ou annulée par ailleurs",
			services.NO_INFORMATION, nil)
		return
	}
	services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
}

func rendreJustification(w http.ResponseWriter, r *http.Request, q *Queries, id int64, status int) {
	row, err := q.GetJustification(context.Background(), id)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}
	render.Status(r, status)
	render.JSON(w, r, getToItem(row))
}

// ─── Lecture ─────────────────────────────────────────────────────────────────

// ListHandler alimente la page de gestion.
// GET /justifications?promo_id=&periode_id=&user_id=&include_revoked=&q=
func ListHandler(w http.ResponseWriter, r *http.Request) {
	recherche := pgtype.Text{}
	if s := r.URL.Query().Get("q"); s != "" {
		recherche = pgtype.Text{String: s, Valid: true}
	}
	includeRevoked, _ := strconv.ParseBool(r.URL.Query().Get("include_revoked"))

	rows, err := New(services.GetPgCtx(r.Context()).Db).ListJustifications(context.Background(),
		ListJustificationsParams{
			PromoID:        queryInt8(r, "promo_id"),
			PeriodeID:      queryInt8(r, "periode_id"),
			UserID:         queryInt4(r, "user_id"),
			Recherche:      recherche,
			IncludeRevoked: includeRevoked,
		})
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	items := make([]justificationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toItem(row))
	}
	render.JSON(w, r, items)
}

// SeancesHandler détaille la couverture d'une justification (dépliage de ligne).
// GET /justifications/{id}/seances
func SeancesHandler(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		services.InvalidRequestError(w, r, "id invalide", services.NO_INFORMATION, nil)
		return
	}

	rows, err := New(services.GetPgCtx(r.Context()).Db).ListJustificationSeances(context.Background(), id)
	if err != nil {
		services.InternalServerError(w, r, err.Error(), services.NO_INFORMATION, nil)
		return
	}

	items := make([]seanceItem, 0, len(rows))
	for _, s := range rows {
		items = append(items, seanceItem{
			ID:           s.ID,
			StartsAt:     fmtTs(s.StartsAt),
			EndsAt:       fmtTs(s.EndsAt),
			Matiere:      s.MatiereName,
			Salle:        s.Salle,
			Statut:       s.Statut,
			DansPlage:    true,
			DejaCouverte: true,
		})
	}
	render.JSON(w, r, items)
}
