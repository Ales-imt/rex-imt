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