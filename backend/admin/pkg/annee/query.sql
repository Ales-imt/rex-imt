-- name: ListAnnee :many
SELECT * FROM public.annee ORDER BY debut DESC;

-- name: GetAnneeById :one
SELECT * FROM public.annee WHERE id = $1;

-- name: CreateAnnee :one
INSERT INTO public.annee (name, debut, fin)
VALUES (@name, @debut, @fin)
RETURNING *;

-- name: UpdateAnnee :one
UPDATE public.annee
SET name = @name,
    debut = @debut,
    fin = @fin
WHERE id = @id
RETURNING *;

-- name: DeleteAnnee :exec
DELETE FROM public.annee WHERE id = @id;

-- name: DeleteAnneeByIds :exec
DELETE FROM public.annee WHERE id = ANY(@ids::bigint[]);
