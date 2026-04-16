-- name: ListFeedbacks :many
SELECT id, content, created_at, promotion
FROM feedback f
LEFT JOIN student s ON f.user_id = s.user_id
ORDER BY f.created_at DESC;