-- name: InsertFeedbacks :one

INSERT INTO feedback (user_id, content, created_at, season_id, strongbox, pseudo, message_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7) returning id;