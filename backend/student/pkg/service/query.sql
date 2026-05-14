-- name: UpsertPromotion :exec
INSERT INTO promotion (id, name)
VALUES (@id, @name)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

-- name: UpsertPeriode :one
INSERT INTO periode (name, promotion_id)
VALUES (@name, @promotion_id)
ON CONFLICT (name, promotion_id) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: UpsertMatiere :exec
INSERT INTO matiere (id, name, periode_id)
VALUES (@id, @name, @periode_id)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, periode_id = EXCLUDED.periode_id;
