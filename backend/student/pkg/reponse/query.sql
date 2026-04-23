-- name: GetUnreadReponsesByPseudo :many
SELECT r.id, r.message_id, r.contenu, r.auteur, r.cree_le, r.lu
FROM reponse r
JOIN feedback f ON r.message_id = f.message_id
WHERE f.pseudo = $1 AND r.lu = false;

-- name: MarkReponsesByPseudoAsRead :exec
UPDATE reponse SET lu = true
WHERE message_id IN (
    SELECT message_id FROM feedback WHERE pseudo = $1
) AND lu = false;
