-- name: UpsertPromotion :exec
INSERT INTO promotion (id, name)
VALUES (@id, @name)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

-- name: UpsertPeriode :one
INSERT INTO periode (name, promotion_id, annee)
VALUES (@name, @promotion_id, @annee)
ON CONFLICT (name, promotion_id, annee) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: UpsertMatiere :exec
INSERT INTO matiere (external_id, name, periode_id, annee)
VALUES (@external_id, @name, @periode_id, @annee)
ON CONFLICT (external_id, annee) DO UPDATE SET name = EXCLUDED.name, periode_id = EXCLUDED.periode_id;
