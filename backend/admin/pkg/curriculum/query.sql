-- name: GetDistinctAnnees :many
SELECT DISTINCT annee FROM matiere ORDER BY annee DESC;

-- name: GetAnneeCourante :one
-- Année académique courante d'après la table annee (matiere.annee = année de
-- début), exactement la même règle que côté étudiant (cf. GetMatieresAEvaluer).
-- Sans elle, l'admin retombe sur max(matiere.annee), qui est l'année à venir dès
-- que son calendrier est chargé : on affiche alors une matière homonyme de
-- l'année suivante, vide de toute évaluation.
-- LIMIT 1 : si deux années se chevauchent, on prend la plus récente.
SELECT EXTRACT(YEAR FROM a.debut)::int AS annee
FROM annee a
WHERE a.debut <= CURRENT_DATE
  AND a.fin >= CURRENT_DATE
ORDER BY a.debut DESC
LIMIT 1;

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
