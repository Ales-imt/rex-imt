-- name: GetPromotionById :one
SELECT * FROM promotion where id = $1;

-- name: CreationPromotion :exec
INSERT INTO promotion (id, name) 
    VALUES ($1, $2) ON CONFLICT (id) DO UPDATE
SET
    name = EXCLUDED.name;

-- name: GetPromotions :many
SELECT * FROM promotion
    ORDER BY name;

-- name: GetGroupe :many
SELECT * FROM groupe
    ORDER BY name;

-- name: CreationGroupe :exec
INSERT INTO groupe (id, name, promo_id) 
    VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE
SET
    name = EXCLUDED.name,
    promo_id = EXCLUDED.promo_id;

-- name: UpdateStudentPromo :exec
UPDATE public.student
SET promotion = @promotion
WHERE user_id = @id;

-- name: AddEleveToGroupe :exec
insert into eleve_groupe(num_etudiant, id_groupe)
    values ($1,$2);

-- name: DeleteEleveToGroupe :exec
delete from eleve_groupe;


