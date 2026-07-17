-- name: GetSeanceByCode :one
SELECT s.id, s.opened_at, s.closed_at, s.late_after_minutes, m.name AS matiere_name
FROM seance s JOIN matiere m ON m.id = s.matiere_id
WHERE s.code = @code;

-- name: GetSeanceByID :one
SELECT s.id, s.opened_at, s.closed_at, s.late_after_minutes, m.name AS matiere_name
FROM seance s JOIN matiere m ON m.id = s.matiere_id
WHERE s.id = @id;

-- name: GetExistingPointage :one
SELECT statut, pointe_at FROM pointage
WHERE seance_id = @seance_id AND user_id = @user_id;

-- name: UpsertPointage :one
INSERT INTO pointage (seance_id, user_id, statut, pointe_at)
VALUES (@seance_id, @user_id, @statut, CURRENT_TIMESTAMP)
ON CONFLICT (seance_id, user_id) DO NOTHING
RETURNING seance_id, user_id, statut, pointe_at;

-- Les requêtes du registre presence_ledger sont dans back-rex-common/pkg/ledger
-- (AppendLedger y est l'unique implémentation du chaînage).
