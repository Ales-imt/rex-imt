-- name: AnonymizeFeedback :execrows
UPDATE feedback SET
    strongbox  = NULL,
    pseudo     = NULL,
    message_id = NULL,
    promotion  = NULL,
    groupe     = NULL
WHERE created_at < $1
  AND pseudo IS NOT NULL;

-- name: AnonymizeClassification :execrows
UPDATE feedback_classification fc SET
    promotion = NULL,
    groupe    = NULL
FROM feedback f
WHERE fc.feedback_id = f.id
  AND f.created_at < $1
  AND (fc.promotion IS NOT NULL OR fc.groupe IS NOT NULL);

-- name: AnonymizeEvalVerbatim :execrows
UPDATE eval_verbatim SET strongbox = NULL
WHERE created_at < $1
  AND strongbox IS NOT NULL;

-- name: DeleteClassification :execrows
DELETE FROM feedback_classification
WHERE feedback_id IN (
    SELECT id FROM feedback WHERE created_at < $1
);

-- name: DeleteFeedback :execrows
DELETE FROM feedback WHERE created_at < $1;

-- name: PurgeRejectedFeedback :execrows
-- Un feedback REJECTED n'a jamais été publié : aucune conservation LCEN ne s'y
-- applique. On supprime la ligne (raw_content + strongbox compris) passé un
-- court délai après la décision de modération.
DELETE FROM feedback
WHERE moderation_status = 'REJECTED'
  AND moderated_at < $1;

-- name: PurgeRejectedVerbatim :execrows
-- Un verbatim REJECTED n'a jamais été publié : aucune conservation LCEN ne s'y
-- applique. On supprime la ligne (raw_texte chiffré + strongbox compris) passé
-- un court délai après la décision de modération.
DELETE FROM eval_verbatim
WHERE moderation_status = 'REJECTED'
  AND moderated_at < $1;

-- name: GetUserIDByEmail :one
SELECT id FROM "user" WHERE email = $1;

-- name: DeleteUserByID :exec
DELETE FROM "user" WHERE id = $1;
