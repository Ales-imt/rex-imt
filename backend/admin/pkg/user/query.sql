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
    name = @name,
    surname = @surname,
    email = @email,
    roles = @roles,
    blame = @blame
WHERE id = @id AND version = @version
RETURNING *;

-- La suppression de comptes ne passe plus par ce fichier : elle est portée par
-- backend/admin/pkg/account, seule définition du cycle de vie (supprimer /
-- anonymiser / conserver), partagée avec la purge RGPD automatique.
--
-- DeleteUser et DeleteUserByIds ont été retirées :
--   - un DELETE sur un compte porteur de présence est refusé par la FK RESTRICT
--     de presence_ledger, et détruirait ses pointages via les FK CASCADE ;
--   - DeleteUserByIds appliquait un DELETE ... WHERE id = ANY(...) global, qui
--     échouait en bloc dès qu'UN seul identifiant portait de la présence.
