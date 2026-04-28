-- name: InsertPostit :one
INSERT INTO postit (message_id, texte, auteur_id, cree_le)
VALUES ($1, $2, $3, now())
RETURNING id, message_id, texte, auteur_id, cree_le;

-- name: ListPostitsWithDetails :many
SELECT p.id, p.message_id, p.texte, p.cree_le,
       u.name    AS auteur_nom,
       u.surname AS auteur_prenom,
       f.content AS feedback_content,
       c.categorie, c.resume, c.sentiment, c.urgence
FROM postit p
JOIN "user"  u ON u.id = p.auteur_id
JOIN feedback f ON f.message_id = p.message_id
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
ORDER BY p.cree_le DESC;
