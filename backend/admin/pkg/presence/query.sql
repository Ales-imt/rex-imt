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

-- name: ActivateSeance :one
UPDATE seance
SET code = @code, opened_at = NOW(), late_after_minutes = @late_after_minutes
WHERE id = @id AND code IS NULL
RETURNING id, code, opened_at;

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

-- name: CloseSeance :exec
UPDATE seance SET closed_at = CURRENT_TIMESTAMP WHERE id = @id AND closed_at IS NULL;

-- name: GetSeance :one
SELECT s.id, s.matiere_id, s.code, s.opened_at, s.closed_at,
       s.late_after_minutes, m.name AS matiere_name
FROM seance s JOIN matiere m ON m.id = s.matiere_id WHERE s.id = @id;

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

-- name: ListPresence :many
SELECT DISTINCT u.id AS user_id, u.name, u.surname,
       COALESCE(p.statut, 'ABSENT')::text AS statut, p.pointe_at
FROM seance s
JOIN matiere m       ON m.id = s.matiere_id
JOIN periode pe      ON pe.id = m.periode_id
JOIN promotion pr    ON pr.id = pe.promotion_id
JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
JOIN eleve_groupe eg ON eg.id_groupe = g.id
JOIN "user" u        ON u.id = eg.num_etudiant
LEFT JOIN pointage p ON p.user_id = u.id AND p.seance_id = s.id
WHERE s.id = @seance_id
ORDER BY u.surname, u.name;

-- name: ListPresenceHorsGroupe :many
SELECT u.id AS user_id, u.name, u.surname,
       p.statut::text AS statut, p.pointe_at
FROM pointage p
JOIN "user" u ON u.id = p.user_id
WHERE p.seance_id = @seance_id
  AND u.id NOT IN (
    SELECT eg.num_etudiant
    FROM seance s
    JOIN matiere m       ON m.id = s.matiere_id
    JOIN periode pe      ON pe.id = m.periode_id
    JOIN promotion pr    ON pr.id = pe.promotion_id
    JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
    JOIN eleve_groupe eg ON eg.id_groupe = g.id
    WHERE s.id = @seance_id
  )
ORDER BY u.surname, u.name;

-- name: ListSeancesByMatiere :many
SELECT id, code, opened_at, closed_at FROM seance
WHERE matiere_id = @matiere_id ORDER BY opened_at DESC;

-- ── Registre d'intégrité ─────────────────────────────────────────────────────

-- name: GetLastLedgerEntry :one
-- Dernier maillon du registre (pour ancrage TSA et démarrage de VerifyChain).
SELECT seq, hash FROM presence_ledger ORDER BY seq DESC LIMIT 1;

-- name: ListLedgerBySeq :many
-- Parcours ordonné pour VerifyChain : recalcule chaque hash et compare.
SELECT seq, seance_id, user_id, statut, event_at, recorded_at, prev_hash, hash
FROM presence_ledger
ORDER BY seq ASC;

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
