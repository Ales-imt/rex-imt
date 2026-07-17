-- name: ListSeancesPlanning :many
SELECT s.id, s.starts_at, s.ends_at,
       COALESCE(s.salle, '') AS salle,
       COALESCE(s.prof, '') AS prof,
       COALESCE(g.name, pr.name, '') AS promo
FROM seance s
LEFT JOIN groupe g     ON g.id = s.groupe_id
LEFT JOIN promotion pr ON pr.id = s.promotion_id
WHERE s.matiere_id = @matiere_id
  AND s.cancelled_at IS NULL
  AND s.starts_at IS NOT NULL
ORDER BY s.starts_at;

-- name: OpenSeance :one
INSERT INTO seance (matiere_id, code, starts_at, ends_at, salle, prof, late_after_minutes)
VALUES (@matiere_id, @code, @starts_at, @ends_at, @salle, @prof, @late_after_minutes)
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
       COALESCE(s.salle, '') AS salle,
       COALESCE(s.prof, '') AS prof,
       m.name AS matiere_name,
       COALESCE(g.name, pr.name, '') AS promo
FROM seance s
JOIN matiere m         ON m.id = s.matiere_id
LEFT JOIN groupe g     ON g.id = s.groupe_id
LEFT JOIN periode pe   ON pe.id = m.periode_id
LEFT JOIN promotion pr ON pr.id = pe.promotion_id
WHERE s.id = @id;

-- name: ListSeancesByMatiere :many
SELECT id, code, opened_at, closed_at FROM seance
WHERE matiere_id = @matiere_id ORDER BY opened_at DESC;

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
