-- name: GetMatiereExternalID :one
SELECT external_id FROM matiere WHERE id = @id;

-- name: OpenSeance :one
INSERT INTO seance (matiere_id, code, starts_at, ends_at, salle, prof, late_after_minutes)
VALUES (@matiere_id, @code, @starts_at, @ends_at, @salle, @prof, @late_after_minutes)
ON CONFLICT (matiere_id, starts_at) DO UPDATE SET code = EXCLUDED.code
RETURNING id, code, opened_at;

-- name: CloseSeance :exec
UPDATE seance SET closed_at = CURRENT_TIMESTAMP WHERE id = @id AND closed_at IS NULL;

-- name: GetSeance :one
SELECT s.id, s.matiere_id, s.code, s.opened_at, s.closed_at,
       s.late_after_minutes, m.name AS matiere_name
FROM seance s JOIN matiere m ON m.id = s.matiere_id WHERE s.id = @id;

-- name: ListPresence :many
SELECT u.id AS user_id, u.name, u.surname,
       COALESCE(p.statut, 'ABSENT')::text AS statut, p.pointe_at
FROM matiere m
JOIN periode pe   ON pe.id = m.periode_id
JOIN promotion pr ON pr.id = pe.promotion_id
JOIN student st   ON st.promotion = pr.name
JOIN "user" u     ON u.id = st.user_id
LEFT JOIN pointage p ON p.user_id = u.id AND p.seance_id = @seance_id
WHERE m.id = @matiere_id
ORDER BY u.surname, u.name;

-- name: ListSeancesByMatiere :many
SELECT id, code, opened_at, closed_at FROM seance
WHERE matiere_id = @matiere_id ORDER BY opened_at DESC;
