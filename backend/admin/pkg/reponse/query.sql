-- name: InsertReponseByFeedbackId :one
INSERT INTO reponse (message_id, contenu, auteur, cree_le, lu)
SELECT f.message_id, $2, $3, now(), false
FROM feedback f
WHERE f.id = $1
RETURNING id, message_id, contenu, auteur, cree_le, lu;

-- name: ListReponsesByFeedbackId :many
SELECT r.id, r.message_id, r.contenu, r.auteur, r.cree_le, r.lu
FROM reponse r
JOIN feedback f ON f.message_id = r.message_id
WHERE f.id = $1
ORDER BY r.cree_le ASC;
