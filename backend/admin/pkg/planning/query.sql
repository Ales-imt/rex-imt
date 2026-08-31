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

-- name: GetOccupationSalles :many
-- Occupation PLANIFIÉE sur [debut, fin[ : heures réservées par salle.
-- LEFT JOIN depuis salle : une salle sans aucune réservation sort à 0h, et
-- c'est le cas le plus intéressant pour la scolarité.
SELECT
    sa.id                                 AS salle_id,
    sa.name                               AS salle_name,
    sa.capacite                           AS capacite,
    sa.type                               AS type,
    COUNT(s.id)::bigint                   AS nb_seances,
    COALESCE(SUM(EXTRACT(EPOCH FROM (s.ends_at - s.starts_at))) / 3600.0, 0)::float8 AS heures
FROM public.salle sa
LEFT JOIN public.seance s
       ON s.salle_id     = sa.id
      AND s.starts_at   >= @debut
      AND s.starts_at    < @fin
      AND s.ends_at IS NOT NULL
      AND s.cancelled_at IS NULL
GROUP BY sa.id, sa.name, sa.capacite, sa.type
ORDER BY heures DESC, sa.name;

-- name: GetCreneauxSalles :many
-- Le détail, pour la vue calendrier.
SELECT
    sa.id       AS salle_id,
    sa.name     AS salle_name,
    s.id        AS seance_id,
    s.starts_at AS starts_at,
    s.ends_at   AS ends_at,
    m.name      AS matiere_name,
    s.prof      AS prof,
    g.name      AS groupe_name,
    p.name      AS promotion_name
FROM public.seance s
JOIN public.salle sa   ON sa.id = s.salle_id
JOIN public.matiere m  ON m.id  = s.matiere_id
LEFT JOIN public.groupe g    ON g.id = s.groupe_id
LEFT JOIN public.promotion p ON p.id = s.promotion_id
WHERE s.starts_at >= @debut AND s.starts_at < @fin
  AND s.ends_at IS NOT NULL
  AND s.cancelled_at IS NULL
ORDER BY sa.name, s.starts_at;

-- name: GetSallesNonResolues :many
-- Séances portant un libellé de salle absent du référentiel. Sert d'écran de
-- supervision : une entrée ici est soit une salle nouvelle en amont, soit un
-- rattachement cassé — dans les deux cas, des heures qui manquent au bilan.
SELECT btrim(s.salle) AS libelle, COUNT(*)::bigint AS nb_seances
FROM public.seance s
WHERE s.starts_at >= @debut AND s.starts_at < @fin
  AND s.cancelled_at IS NULL
  AND s.salle_id IS NULL
  AND btrim(COALESCE(s.salle, '')) <> ''
GROUP BY 1
ORDER BY nb_seances DESC;

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
