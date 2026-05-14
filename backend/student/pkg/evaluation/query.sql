-- name: InsertEvalSession :one
INSERT INTO eval_session (
    id,
    anon_id,
    matiere_id,
    format_suivi,
    tendance_semestre,
    assiduite,
    created_at
) VALUES (
    gen_random_uuid(),
    @anon_id,
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


-- name: GetSubmittedMatiereIDs :many
SELECT es.matiere_id
FROM eval_session es
WHERE es.anon_id = @anon_id
  AND es.submitted_at IS NOT NULL;


-- name: EvalSessionExists :one
SELECT EXISTS (
    SELECT 1
    FROM eval_session
    WHERE anon_id  = @anon_id
      AND matiere_id = @matiere_id
) AS already_submitted;