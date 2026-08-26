-- name: GetReservationsByPeriode :many
SELECT
    s.id         AS id,
    s.starts_at  AS starts_at,
    s.ends_at    AS ends_at,
    s.matiere_id AS matiere_id,
    m.name       AS matiere_name,
    s.salle      AS salle,
    s.prof       AS prof,
    s.prof_id    AS prof_id,
    s.groupe_id  AS groupe_id,
    g.name       AS groupe_name,
    s.remarque   AS remarque
FROM seance s
JOIN matiere m ON m.id = s.matiere_id
LEFT JOIN groupe g ON g.id = s.groupe_id
WHERE m.periode_id = $1
  AND s.starts_at IS NOT NULL
  AND s.ends_at IS NOT NULL
  AND s.cancelled_at IS NULL
ORDER BY s.starts_at;

-- name: GetHeuresConsommeesByPeriode :many
SELECT
    m.id   AS matiere_id,
    m.name AS matiere_name,
    g.id   AS groupe_id,
    g.name AS groupe_name,
    COALESCE(SUM(EXTRACT(EPOCH FROM (s.ends_at - s.starts_at))) / 3600.0, 0)::float8 AS heures_consommees
FROM seance s
JOIN matiere m ON m.id = s.matiere_id
LEFT JOIN groupe g ON g.id = s.groupe_id
WHERE m.periode_id = $1
  AND s.starts_at IS NOT NULL
  AND s.ends_at IS NOT NULL
  AND s.cancelled_at IS NULL
GROUP BY m.id, m.name, g.id, g.name
ORDER BY m.name, g.name NULLS LAST;

-- name: GetHeuresConsommeesByGroupe :many
SELECT
    g.id   AS groupe_id,
    g.name AS groupe_name,
    m.id   AS matiere_id,
    m.name AS matiere_name,
    COALESCE(SUM(EXTRACT(EPOCH FROM (s.ends_at - s.starts_at))) / 3600.0, 0)::float8 AS heures_consommees
FROM seance s
JOIN matiere m ON m.id = s.matiere_id
LEFT JOIN groupe g ON g.id = s.groupe_id
WHERE m.periode_id = $1
  AND s.starts_at IS NOT NULL
  AND s.ends_at IS NOT NULL
  AND s.cancelled_at IS NULL
GROUP BY g.id, g.name, m.id, m.name
ORDER BY g.name NULLS LAST, m.name;

-- name: GetHeuresConsommeesByProf :many
SELECT
    s.prof_id AS prof_id,
    s.prof    AS prof,
    COALESCE(SUM(EXTRACT(EPOCH FROM (s.ends_at - s.starts_at))) / 3600.0, 0)::float8 AS heures_consommees
FROM seance s
JOIN matiere m ON m.id = s.matiere_id
WHERE m.periode_id = $1
  AND s.starts_at IS NOT NULL
  AND s.ends_at IS NOT NULL
  AND s.cancelled_at IS NULL
GROUP BY s.prof_id, s.prof
ORDER BY s.prof NULLS LAST;
