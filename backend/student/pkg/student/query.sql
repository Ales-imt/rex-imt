-- name: GetStudentElo :one
SELECT elo 
FROM student 
WHERE user_id = $1;

-- name: GetStudentRank :one
SELECT rank 
FROM (
	SELECT user_id, RANK() OVER (ORDER BY elo DESC) AS rank FROM student
	) r WHERE user_id = $1;

-- name: GetStudentLastEloGain :one
SELECT last_elo_gain
FROM student 
WHERE user_id = $1;

-- name: UpdateStudentElo :exec
UPDATE student 
SET elo = elo + $1, last_elo_gain = $1
WHERE user_id = $2;

-- name: ClearLastEloGain :exec
UPDATE student
SET last_elo_gain = 0 
WHERE user_id = $1;




