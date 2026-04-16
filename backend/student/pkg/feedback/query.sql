-- name: InsertFeedbacks :one

INSERT INTO feedback (user_id, content, created_at, season_id)
			VALUES ($1, $2, $3, $4) returning id;