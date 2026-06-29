-- name: GetDistinctAnnees :many
SELECT DISTINCT annee FROM matiere ORDER BY annee DESC;

-- name: GetAllPromotionTree :many
SELECT
    pr.id   AS promotion_id,
    pr.name AS promotion_name,
    pe.id   AS periode_id,
    pe.name AS periode_name,
    m.id    AS matiere_id,
    m.name  AS matiere_name,
    COUNT(es.id) FILTER (WHERE es.submitted_at IS NOT NULL)::int AS nb_repondants
FROM promotion pr
JOIN periode pe ON pe.promotion_id = pr.id
JOIN matiere m ON m.periode_id = pe.id
LEFT JOIN eval_session es ON es.matiere_id = m.id
WHERE m.annee = $1
GROUP BY pr.id, pr.name, pe.id, pe.name, m.id, m.name
ORDER BY pr.name, pe.name, m.name;
