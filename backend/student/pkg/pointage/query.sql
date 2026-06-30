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

-- name: GetLastLedgerHash :one
-- Retourne le hash du dernier maillon du registre pour construire la chaîne.
-- DOIT être exécuté dans la même transaction que InsertLedgerEntry,
-- après pg_advisory_xact_lock, pour garantir la sérialisation.
SELECT hash FROM presence_ledger ORDER BY seq DESC LIMIT 1;

-- name: InsertLedgerEntry :one
-- Insère un maillon dans le registre. recorded_at est fourni explicitement
-- (pas de DEFAULT) pour être connu avant le calcul du hash.
INSERT INTO presence_ledger (seance_id, user_id, statut, event_at, recorded_at, prev_hash, hash)
VALUES (@seance_id, @user_id, @statut, @event_at, @recorded_at, @prev_hash, @hash)
RETURNING seq;
