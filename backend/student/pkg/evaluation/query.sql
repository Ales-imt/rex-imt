-- name: InsertEvalSession :one
INSERT INTO eval_session (
    id,
    pseudo,
    matiere_id,
    format_suivi,
    tendance_semestre,
    assiduite,
    created_at
) VALUES (
    gen_random_uuid(),
    @pseudo,
    @matiere_id,
    @format_suivi,
    @tendance_semestre,
    @assiduite,
    CURRENT_TIMESTAMP
)
RETURNING id;


-- name: InsertEvalScores :exec
INSERT INTO eval_scores (
    session_id,
    score_peda_global,
    score_peda_clarte,
    score_contenu_adequation,
    charge_perception,
    charge_heures_semaine,
    score_difficulte,
    score_supports,
    score_ambiance,
    nps
) VALUES (
    @session_id,
    @score_peda_global,
    @score_peda_clarte,
    @score_contenu_adequation,
    @charge_perception,
    @charge_heures_semaine,
    @score_difficulte,
    @score_supports,
    @score_ambiance,
    @nps
);


-- name: InsertEvalChipReponse :exec
INSERT INTO eval_chip_reponse (
    session_id,
    chip_id
) VALUES (
    @session_id,
    @chip_id
);


-- name: InsertEvalVerbatim :exec
INSERT INTO eval_verbatim (
    id,
    session_id,
    dimension,
    texte,
    created_at,
    strongbox
) VALUES (
    gen_random_uuid(),
    @session_id,
    @dimension,
    @texte,
    CURRENT_TIMESTAMP,
    @strongbox
);


-- name: SubmitEvalSession :exec
UPDATE eval_session
SET submitted_at = CURRENT_TIMESTAMP
WHERE id = @session_id;


-- name: GetMatieresAEvaluer :many
-- Matières de l'étudiant pour l'année académique courante dont toutes les
-- séances sont terminées (aucune séance non annulée dans le futur).
-- L'identifiant renvoyé est l'id interne de matiere.
SELECT
    m.id::text AS matiere_id,
    m.name::text AS nom,
    COALESCE((array_agg(s.prof ORDER BY s.ends_at DESC))[1], '')::text AS prof,
    COALESCE((array_agg(COALESCE(gs.name, pr.name) ORDER BY s.ends_at DESC))[1], '')::text AS formation
FROM seance s
JOIN matiere m ON m.id = s.matiere_id
JOIN periode pe ON pe.id = m.periode_id
JOIN promotion pr ON pr.id = pe.promotion_id
LEFT JOIN groupe gs ON gs.id = s.groupe_id
WHERE m.annee = (
      -- Année académique courante d'après la table annee (matiere.annee = année de debut).
      -- LIMIT 1 : si plusieurs périodes se chevauchent, on prend la plus récente.
      SELECT EXTRACT(YEAR FROM a.debut)::int
      FROM annee a
      WHERE a.debut <= CURRENT_DATE AND a.fin >= CURRENT_DATE
      ORDER BY a.debut DESC
      LIMIT 1
  )
  AND s.cancelled_at IS NULL
  AND EXISTS (
      SELECT 1
      FROM eleve_groupe eg
      JOIN groupe g ON g.id = eg.id_groupe
      WHERE eg.num_etudiant = @user_id
        AND g.promo_id = pr.id
        AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
  )
GROUP BY m.id, m.name
HAVING COUNT(*) FILTER (WHERE s.ends_at IS NULL OR s.ends_at > NOW()) = 0
ORDER BY m.name;


-- name: GetSubmittedMatiereIDs :many
SELECT es.matiere_id::text
FROM eval_session es
WHERE es.pseudo = @pseudo
  AND es.submitted_at IS NOT NULL;


-- name: EvalSessionExists :one
SELECT EXISTS (
    SELECT 1
    FROM eval_session
    WHERE pseudo = @pseudo
      AND matiere_id = @matiere_id
) AS already_submitted;


-- name: GetSessionByMatiere :one
SELECT
    es.id,
    es.format_suivi,
    es.tendance_semestre,
    es.assiduite,
    sc.score_peda_global,
    sc.score_peda_clarte,
    sc.score_contenu_adequation,
    sc.charge_perception,
    sc.charge_heures_semaine,
    sc.score_difficulte,
    sc.score_supports,
    sc.score_ambiance,
    sc.nps
FROM eval_session es
JOIN eval_scores sc ON sc.session_id = es.id
WHERE es.pseudo = @pseudo
  AND es.matiere_id = @matiere_id
  AND es.submitted_at IS NOT NULL;


-- name: GetMatiereIDAnneeCourante :one
-- Vérifie que la matière existe pour l'année académique courante,
-- déterminée par la table annee (matiere.annee = année de debut).
SELECT m.id FROM matiere m
WHERE m.id = @matiere_id
  AND m.annee = (
      SELECT EXTRACT(YEAR FROM a.debut)::int
      FROM annee a
      WHERE a.debut <= CURRENT_DATE AND a.fin >= CURRENT_DATE
      ORDER BY a.debut DESC
      LIMIT 1
  );

-- name: DeleteEvalSession :exec
DELETE FROM eval_session WHERE id = @session_id AND pseudo = @pseudo;