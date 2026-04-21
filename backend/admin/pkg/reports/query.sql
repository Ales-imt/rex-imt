-- name: ReportGlobalStats :one
SELECT
    COUNT(*)                        AS total,
    COUNT(c.feedback_id)            AS classified,
    COUNT(*) - COUNT(c.feedback_id) AS pending
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE f.created_at >= $1;

-- name: ReportByUrgence :many
SELECT c.urgence::text AS label, COUNT(*) AS count
FROM feedback f
JOIN feedback_classification c ON c.feedback_id = f.id
WHERE f.created_at >= $1
GROUP BY c.urgence
ORDER BY c.urgence DESC;

-- name: ReportByCategorie :many
SELECT COALESCE(c.categorie, 'Non classifié') AS label, COUNT(*) AS count
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE f.created_at >= $1
GROUP BY c.categorie
ORDER BY count DESC;

-- name: ReportBySentiment :many
SELECT COALESCE(c.sentiment, 'non classifié') AS label, COUNT(*) AS count
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE f.created_at >= $1
GROUP BY c.sentiment
ORDER BY count DESC;

-- name: ReportByPromo :many
SELECT
    COALESCE(c.promotion, 'Inconnue') AS promo,
    COUNT(*)                          AS count,
    COALESCE(MAX(c.urgence), 0)       AS max_urgence
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE f.created_at >= $1
GROUP BY c.promotion
ORDER BY count DESC;
