-- name: GetStudentPromoAndGroupes :one
SELECT
    p.name                                     AS promotion,
    string_agg(g.name, ', ' ORDER BY g.name)::text  AS groupes
FROM eleve_groupe eg
JOIN groupe    g ON eg.id_groupe   = g.id
JOIN promotion p ON g.promo_id     = p.id
WHERE eg.num_etudiant = $1
GROUP BY p.name
LIMIT 1;

-- name: InsertFeedbacks :one

INSERT INTO feedback (content, created_at, strongbox, pseudo, message_id, promotion, groupe)
			VALUES ($1, $2, $3, $4, $5, $6, $7) returning id;

-- name: AnonymizeFeedback :exec
UPDATE feedback
SET
  content   = '[contenu supprimé à la demande de l''auteur]',
  pseudo    = NULL,
  promotion = NULL,
  groupe    = NULL
WHERE message_id = $1
  AND pseudo = $2;

-- name: AnonymizePostitByFeedback :exec
UPDATE postit SET message_modere = '[contenu supprimé à la demande de l''auteur]'
WHERE message_id = $1;