-- name: GetCurrentSeason :one
SELECT id
FROM season
WHERE start_date <= $1 AND end_date > $1
LIMIT 1;

-- name: CreateSeason :one
INSERT INTO season (start_date, end_date)
VALUES ($1, $2)
RETURNING id;

-- name: GetSeasonEndDate :one
SELECT end_date FROM season WHERE id = $1;