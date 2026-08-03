-- name: GetEvalStatsByMatiere :one
SELECT
    COUNT(DISTINCT es.id) FILTER (WHERE es.submitted_at IS NOT NULL)::int                                       AS nb_repondants,
    AVG(sc.score_peda_global)::float8                                                                           AS score_peda,
    AVG(sc.score_peda_clarte)::float8                                                                           AS score_clarte,
    AVG(sc.score_contenu_adequation)::float8                                                                    AS score_contenu,
    AVG(sc.score_difficulte)::float8                                                                            AS score_difficulte,
    AVG(sc.score_supports)::float8                                                                              AS score_supports,
    AVG(sc.score_ambiance)::float8                                                                              AS score_ambiance,
    AVG(sc.nps)::float8                                                                                         AS nps_moyen,
    COALESCE(COUNT(*) FILTER (WHERE sc.charge_perception = 'LOURDE') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_charge_lourde,
    COALESCE(COUNT(*) FILTER (WHERE sc.charge_perception = 'LEGERE') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_charge_legere,
    COALESCE(COUNT(*) FILTER (WHERE es.format_suivi = 'PRESENTIEL') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8  AS pct_format_presentiel,
    COALESCE(COUNT(*) FILTER (WHERE es.format_suivi = 'DISTANCIEL') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8  AS pct_format_distanciel,
    COALESCE(COUNT(*) FILTER (WHERE es.format_suivi = 'HYBRIDE') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8     AS pct_format_hybride,
    COALESCE(COUNT(*) FILTER (WHERE es.tendance_semestre = 'BAISSE') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_tendance_baisse,
    COALESCE(COUNT(*) FILTER (WHERE es.tendance_semestre = 'STABLE') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_tendance_stable,
    COALESCE(COUNT(*) FILTER (WHERE es.tendance_semestre = 'PROGRES') * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_tendance_progres
FROM eval_session es
LEFT JOIN eval_scores sc ON sc.session_id = es.id
WHERE es.matiere_id = $1
  AND es.submitted_at IS NOT NULL;

-- name: GetChipStatsByMatiere :many
SELECT
    cr.id       AS chip_id,
    cr.libelle,
    cr.dimension,
    cr.polarite,
    COUNT(ecr.session_id)::int AS nb,
    COALESCE(
        COUNT(ecr.session_id) * 100.0 / NULLIF(
            (SELECT COUNT(*) FROM eval_session es2 WHERE es2.matiere_id = $1 AND es2.submitted_at IS NOT NULL), 0
        ), 0
    )::float8 AS pct
FROM eval_chip_ref cr
LEFT JOIN eval_chip_reponse ecr ON ecr.chip_id = cr.id
    AND ecr.session_id IN (
        SELECT es3.id FROM eval_session es3 WHERE es3.matiere_id = $1 AND es3.submitted_at IS NOT NULL
    )
WHERE cr.actif = true
GROUP BY cr.id, cr.libelle, cr.dimension, cr.polarite
HAVING COUNT(ecr.session_id) > 0
ORDER BY cr.dimension, cr.polarite DESC, nb DESC;

-- name: GetVerbatimsByMatiere :many
-- Seuls les verbatims PUBLISHED sont affichés : un verbatim PENDING n'a pas
-- encore été relu, un REJECTED a son texte chiffré (age) dans raw_texte et ne
-- doit jamais ressortir. texte est NULLABLE (renseigné à la publication) : on
-- le coalesce pour conserver le type string côté Go.
SELECT ev.id, ev.session_id, ev.dimension,
       COALESCE(ev.texte, '')::text AS texte,
       ev.created_at
FROM eval_verbatim ev
JOIN eval_session es ON es.id = ev.session_id
WHERE es.matiere_id = $1
  AND es.submitted_at IS NOT NULL
  AND ev.moderation_status = 'PUBLISHED'
ORDER BY ev.dimension, ev.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetVerbatimsByMatiereFiltered :many
-- Idem GetVerbatimsByMatiere, restreint à une dimension.
SELECT ev.id, ev.session_id, ev.dimension,
       COALESCE(ev.texte, '')::text AS texte,
       ev.created_at
FROM eval_verbatim ev
JOIN eval_session es ON es.id = ev.session_id
WHERE es.matiere_id = $1
  AND es.submitted_at IS NOT NULL
  AND ev.dimension = $2
  AND ev.moderation_status = 'PUBLISHED'
ORDER BY ev.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetLastSyntheseByMatiere :one
SELECT id, matiere_id, semestre, nb_repondants, synthese_texte, statut, generated_at, validated_by, validated_at
FROM eval_synthese_ia
WHERE matiere_id = $1
ORDER BY generated_at DESC
LIMIT 1;

-- name: GetNpsDistributionByMatiere :one
SELECT
    COALESCE(COUNT(*) FILTER (WHERE sc.nps BETWEEN 0 AND 6)  * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_detracteurs,
    COALESCE(COUNT(*) FILTER (WHERE sc.nps BETWEEN 7 AND 8)  * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_passifs,
    COALESCE(COUNT(*) FILTER (WHERE sc.nps BETWEEN 9 AND 10) * 100.0 / NULLIF(COUNT(*), 0), 0)::float8 AS pct_promoteurs
FROM eval_session es
JOIN eval_scores sc ON sc.session_id = es.id
WHERE es.matiere_id = $1
  AND es.submitted_at IS NOT NULL
  AND sc.nps IS NOT NULL;

-- name: GetEvalDataForPrompt :one
SELECT
    m.name                                                          AS matiere_name,
    pr.name                                                         AS promotion_name,
    pe.name                                                         AS periode_name,
    COUNT(DISTINCT es.id) FILTER (WHERE es.submitted_at IS NOT NULL)::int AS nb_repondants,
    AVG(sc.score_peda_global)::float8                               AS score_peda,
    AVG(sc.score_peda_clarte)::float8                               AS score_clarte,
    AVG(sc.score_contenu_adequation)::float8                        AS score_contenu,
    AVG(sc.score_supports)::float8                                  AS score_supports,
    AVG(sc.score_ambiance)::float8                                  AS score_ambiance,
    AVG(sc.nps)::float8                                             AS nps_moyen
FROM matiere m
JOIN periode pe ON pe.id = m.periode_id
JOIN promotion pr ON pr.id = pe.promotion_id
LEFT JOIN eval_session es ON es.matiere_id = m.id
LEFT JOIN eval_scores sc ON sc.session_id = es.id AND es.submitted_at IS NOT NULL
WHERE m.id = $1
GROUP BY m.name, pr.name, pe.name;

-- name: GetChipsForPrompt :many
SELECT cr.libelle, cr.polarite, COUNT(ecr.session_id)::int AS nb
FROM eval_chip_ref cr
JOIN eval_chip_reponse ecr ON ecr.chip_id = cr.id
    AND ecr.session_id IN (
        SELECT id FROM eval_session WHERE matiere_id = $1 AND submitted_at IS NOT NULL
    )
WHERE cr.actif = true
GROUP BY cr.id, cr.libelle, cr.polarite
ORDER BY cr.polarite DESC, nb DESC;

-- name: GetVerbatimsForPrompt :many
-- L'IA ne reçoit que des verbatims modérés/publiés — même règle que pour les
-- feedbacks, où le déclenchement IA est câblé sur le passage à PUBLISHED
-- (cf. trigger feedback_notify).
SELECT ev.dimension, COALESCE(ev.texte, '')::text AS texte
FROM eval_verbatim ev
JOIN eval_session es ON es.id = ev.session_id
WHERE es.matiere_id = $1
  AND es.submitted_at IS NOT NULL
  AND ev.moderation_status = 'PUBLISHED'
ORDER BY ev.dimension, ev.created_at DESC
LIMIT $2;

-- name: InsertSynthese :one
INSERT INTO eval_synthese_ia (id, matiere_id, semestre, nb_repondants, synthese_texte, statut, generated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, 'GENEREE', NOW())
RETURNING id, matiere_id, semestre, nb_repondants, synthese_texte, statut, generated_at, validated_by, validated_at;

-- name: UpdateSyntheseStatut :exec
UPDATE eval_synthese_ia
SET statut       = $2,
    validated_by = $3,
    validated_at = CASE WHEN $2 = 'VALIDEE' THEN NOW() ELSE NULL END
WHERE id = $1;

