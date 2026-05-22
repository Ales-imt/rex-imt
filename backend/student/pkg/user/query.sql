-- name: GetMe :one
SELECT id, name, surname, email, roles, blame, informed_at
FROM "user"
WHERE id = $1;

-- name: SetInformedAt :exec
UPDATE "user"
SET informed_at = NOW()
WHERE id = $1;
