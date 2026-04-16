-- name: ListUsers :many
SELECT * FROM "user"
ORDER BY id ASC;

-- name: GetUserById :one
SELECT * FROM public.user WHERE id = $1;

-- name: GetUserByMail :one
SELECT * FROM public.user WHERE lower(email) = lower($1);

-- name: ListUser :many
SELECT * FROM public.user ORDER BY id;

-- name: UpdatePartialUser :one
UPDATE public.user
SET version = version + 1,
    roles = @roles,
    blame = @blame
WHERE id = @id AND version = @version
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM public.user WHERE id = @id;
