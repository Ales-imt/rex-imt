-- name: InsertFeedbacks :one

INSERT INTO feedback (content, created_at, strongbox, pseudo, message_id, promotion, groupe)
			VALUES ($1, $2, $3, $4, $5, $6, $7) returning id;