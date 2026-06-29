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


-- name: GetSubmittedMatiereIDs :many
SELECT mm.external_id
FROM eval_session es
JOIN migration.matiere_map mm ON mm.internal_id = es.matiere_id AND mm.source = 'webdfd'
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


-- name: GetMatiereIDByWebdfdAndAnnee :one
SELECT m.id FROM matiere m
JOIN migration.matiere_map mm ON mm.internal_id = m.id
WHERE mm.source = 'webdfd' AND mm.external_id = @webdfd_id AND m.annee = @annee;

-- name: DeleteEvalSession :exec
DELETE FROM eval_session WHERE id = @session_id AND pseudo = @pseudo;