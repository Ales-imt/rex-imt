-- name: ListSeancesPlanning :many
SELECT s.id, s.starts_at, s.ends_at,
       COALESCE(sa.name, '') AS salle,
       COALESCE(s.prof, '') AS prof,
       COALESCE(g.name, pr.name, '') AS promo
FROM seance s
LEFT JOIN salle sa     ON sa.id = s.salle_id
LEFT JOIN groupe g     ON g.id = s.groupe_id
LEFT JOIN promotion pr ON pr.id = s.promotion_id
WHERE s.matiere_id = @matiere_id
  AND s.cancelled_at IS NULL
  AND s.starts_at IS NOT NULL
ORDER BY s.starts_at;

-- name: OpenSeance :one
-- Pas de salle : cette branche ne s'exécute que hors créneau planifié, où
-- l'utilisateur n'en a jamais désigné — et une salle se désigne par sa clé,
-- pas par un texte libre.
INSERT INTO seance (matiere_id, code, starts_at, ends_at, prof, late_after_minutes)
VALUES (@matiere_id, @code, @starts_at, @ends_at, @prof, @late_after_minutes)
ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
RETURNING id, code, opened_at;

-- name: GetSeanceBySlot :one
SELECT id, code, opened_at, closed_at, late_after_minutes
FROM seance
WHERE matiere_id = @matiere_id AND starts_at = @starts_at
ORDER BY opened_at DESC
LIMIT 1;

-- name: GetSeanceDetail :one
SELECT s.id, s.code, s.opened_at, s.closed_at, s.late_after_minutes,
       s.starts_at, s.ends_at,
       COALESCE(sa.name, '') AS salle,
       COALESCE(s.prof, '') AS prof,
       m.name AS matiere_name,
       COALESCE(g.name, pr.name, '') AS promo
FROM seance s
JOIN matiere m         ON m.id = s.matiere_id
LEFT JOIN salle sa     ON sa.id = s.salle_id
LEFT JOIN groupe g     ON g.id = s.groupe_id
LEFT JOIN periode pe   ON pe.id = m.periode_id
LEFT JOIN promotion pr ON pr.id = pe.promotion_id
WHERE s.id = @id;

-- name: ListSeancesByMatiere :many
SELECT id, code, opened_at, closed_at FROM seance
WHERE matiere_id = @matiere_id ORDER BY opened_at DESC;

-- ── Export semestriel ────────────────────────────────────────────────────────

-- name: GetPeriodeHeader :one
-- En-tête du document : interrogée séparément de ListSeancesByPeriode car le
-- document doit s'identifier (et se nommer) même sans aucune séance éditable.
SELECT pe.id, pe.name AS periode_name, pe.annee,
       COALESCE(pr.name, '') AS promo_name
FROM periode pe
LEFT JOIN promotion pr ON pr.id = pe.promotion_id
WHERE pe.id = @id;

-- name: ListSeancesByPeriode :many
-- Toutes les séances d'un semestre en une requête (pas de boucle sur
-- GetSeanceDetail). Les filtres sont optionnels : un tableau ou un horodatage
-- NULL désactive la clause correspondante.
SELECT s.id, s.code, s.opened_at, s.closed_at, s.late_after_minutes,
       s.starts_at, s.ends_at,
       COALESCE(sa.name, '') AS salle,
       COALESCE(s.prof, '') AS prof,
       m.id AS matiere_id,
       m.name AS matiere_name,
       COALESCE(g.name, pr.name, '') AS promo
FROM seance s
JOIN matiere m         ON m.id = s.matiere_id
JOIN periode pe        ON pe.id = m.periode_id
LEFT JOIN salle sa     ON sa.id = s.salle_id
LEFT JOIN promotion pr ON pr.id = pe.promotion_id
LEFT JOIN groupe g     ON g.id = s.groupe_id
WHERE pe.id = @periode_id
  AND s.cancelled_at IS NULL
  AND s.starts_at IS NOT NULL
  AND (@matiere_ids::bigint[] IS NULL OR m.id = ANY(@matiere_ids::bigint[]))
  AND (@from_ts::timestamptz IS NULL OR s.starts_at >= @from_ts::timestamptz)
  AND (@to_ts::timestamptz IS NULL OR s.starts_at < @to_ts::timestamptz)
ORDER BY m.name, s.starts_at;

-- name: ListMatieresPeriodeSummary :many
-- Alimente le dialogue d'export côté web : par matière, le nombre de séances
-- éditables, celles restées ouvertes et l'étendue des dates.
SELECT m.id, m.name,
       COUNT(s.id)::bigint AS nb_seances,
       (COUNT(s.id) FILTER (WHERE s.closed_at IS NULL))::bigint AS nb_non_cloturees,
       MIN(s.starts_at)::timestamptz AS premiere,
       MAX(s.starts_at)::timestamptz AS derniere
FROM matiere m
LEFT JOIN seance s ON s.matiere_id = m.id
                  AND s.cancelled_at IS NULL
                  AND s.starts_at IS NOT NULL
WHERE m.periode_id = @periode_id
GROUP BY m.id, m.name
ORDER BY m.name;

-- name: CountElevesPeriode :one
-- Effectif de la promotion, pour l'estimation du nombre de pages.
SELECT COUNT(DISTINCT eg.num_etudiant)::bigint AS nb
FROM periode pe
JOIN promotion pr    ON pr.id = pe.promotion_id
JOIN groupe g        ON g.promo_id = pr.id
JOIN eleve_groupe eg ON eg.id_groupe = g.id
WHERE pe.id = @periode_id;

-- name: ListLedgerBySeances :many
-- Annexe registre d'intégrité : bornes et empreinte finale du registre pour
-- chaque séance exportée. Le rapprochement avec les ancres TSA se fait en Go
-- (cf. ListAnchorSeqs) pour garder cette requête agrégée simple.
SELECT l.seance_id,
       MIN(l.seq)::bigint AS seq_min,
       MAX(l.seq)::bigint AS seq_max,
       COUNT(*)::bigint   AS nb_maillons,
       (ARRAY_AGG(l.hash ORDER BY l.seq DESC))[1]::text AS dernier_hash
FROM presence_ledger l
WHERE l.seance_id = ANY(@seance_ids::bigint[])
GROUP BY l.seance_id
ORDER BY l.seance_id;

-- name: ListAnchorSeqs :many
-- Repères d'ancrage allégés (sans les jetons) : pour un maillon donné, la
-- première ancre de seq supérieure ou égale le scelle par chaînage.
SELECT ledger_seq, MIN(created_at)::timestamptz AS created_at
FROM presence_anchor
GROUP BY ledger_seq
ORDER BY ledger_seq ASC;

-- ── Registre d'intégrité ─────────────────────────────────────────────────────
-- Les requêtes du registre presence_ledger sont dans back-rex-common/pkg/ledger.

-- name: InsertAnchor :one
-- Archive un jeton RFC 3161 pour un maillon donné.
INSERT INTO presence_anchor (ledger_seq, anchored_hash, tsa_url, hash_algorithm, token, tsa_cert)
VALUES (@ledger_seq, @anchored_hash, @tsa_url, @hash_algorithm, @token, @tsa_cert)
RETURNING id;

-- name: ListAnchors :many
-- Retourne toutes les ancres pour VerifyAnchors.
SELECT id, ledger_seq, anchored_hash, tsa_url, hash_algorithm, token, tsa_cert, created_at
FROM presence_anchor
ORDER BY ledger_seq ASC, id ASC;

-- name: GetAnchorByLedgerSeq :many
-- Ancres existantes pour un maillon donné (idempotence d'AnchorLast).
SELECT id, tsa_url FROM presence_anchor WHERE ledger_seq = @ledger_seq;

-- ── Témoin externe ───────────────────────────────────────────────────────────

-- name: GetAnchorByID :one
-- Charge une ancre pour construire son témoin (envoi initial ou renvoi).
SELECT id, ledger_seq, anchored_hash, tsa_url, token, tsa_cert, created_at
FROM presence_anchor WHERE id = @id;

-- name: HasSentWitness :one
-- Idempotence : un témoin déjà SENT pour (ancre, destinataire) n'est pas renvoyé.
SELECT EXISTS(
  SELECT 1 FROM presence_witness
  WHERE anchor_id = @anchor_id AND recipient = @recipient AND status = 'SENT'
) AS sent;

-- name: InsertWitness :one
-- Trace chaque tentative d'envoi (SENT ou FAILED) pour audit et renvoi.
INSERT INTO presence_witness (anchor_id, ledger_seq, recipient, status, error)
VALUES (@anchor_id, @ledger_seq, @recipient, @status, @error)
RETURNING id;
