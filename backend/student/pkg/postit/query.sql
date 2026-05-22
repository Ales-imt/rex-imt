-- name: ListPostitsPromotions :many
SELECT DISTINCT f.promotion
FROM postit p
JOIN feedback f ON f.message_id = p.message_id
WHERE f.promotion IS NOT NULL AND f.promotion != ''
ORDER BY f.promotion;

-- name: ListPostitsWithDetails :many
SELECT p.id, p.reponse, p.message_modere, p.cree_le,
       u.name    AS auteur_nom,
       u.surname AS auteur_prenom,
       c.categorie, c.resume, c.sentiment, c.urgence
FROM postit p
JOIN "user"  u ON u.id = p.auteur_id
JOIN feedback f ON f.message_id = p.message_id
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE p.cree_le >= now() - (sqlc.arg(months)::int * INTERVAL '1 month')
  AND (sqlc.arg(promotion)::text = '' OR f.promotion = sqlc.arg(promotion))
ORDER BY p.cree_le DESC;
