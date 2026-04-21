-- name: InsertFeedbacks :one

INSERT INTO feedback (user_id, content, created_at, season_id, strongbox)
			VALUES ($1, $2, $3, $4, $5) returning id;