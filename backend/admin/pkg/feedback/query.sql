-- name: ListFeedbacks :many
-- content est devenu NULLABLE (renseigné seulement à la publication) ; on le
-- coalesce pour conserver le type string côté Go et la forme JSON existante.
SELECT id, COALESCE(content, '')::text AS content, created_at
FROM feedback f
ORDER BY f.created_at DESC;

-- name: ListFeedbacksWithClassification :many
SELECT f.id, COALESCE(f.content, '')::text AS content, f.created_at,
       c.promotion, c.groupe,
       c.categorie, c.sous_categorie, c.sentiment, c.urgence, c.resume,
       f.strongbox, f.pseudo, f.message_id
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE f.moderation_status = 'PUBLISHED'
ORDER BY f.created_at DESC;

-- name: ListPendingFeedbacks :many
SELECT f.id, COALESCE(f.content, '')::text AS content, f.promotion, f.groupe
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE c.feedback_id IS NULL
  AND f.moderation_status = 'PUBLISHED'
ORDER BY f.created_at DESC;

-- name: ListRecentFeedbacksWithClassification :many
SELECT f.id, COALESCE(f.content, '')::text AS content, f.created_at,
       c.promotion, c.groupe,
       c.categorie, c.sous_categorie, c.sentiment, c.urgence, c.resume,
       f.strongbox, f.pseudo, f.message_id
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE f.moderation_status = 'PUBLISHED'
  AND f.created_at >= now() - ($1 * interval '1 month')
ORDER BY f.created_at DESC;

-- name: InsertClassification :exec
INSERT INTO feedback_classification (feedback_id, categorie, sous_categorie, sentiment, urgence, resume, promotion, groupe)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (feedback_id) DO UPDATE SET
    categorie      = EXCLUDED.categorie,
    sous_categorie = EXCLUDED.sous_categorie,
    sentiment      = EXCLUDED.sentiment,
    urgence        = EXCLUDED.urgence,
    resume         = EXCLUDED.resume,
    promotion      = EXCLUDED.promotion,
    groupe         = EXCLUDED.groupe,
    classified_at  = now();

-- name: ListenNewFeedback :exec
-- Abonnement au canal alimenté par le trigger feedback_notify (déclenché au
-- passage d'un feedback à PUBLISHED). À exécuter sur une connexion DÉDIÉE et
-- persistante (pas sur le pool) : l'abonnement est lié à la session.
LISTEN new_feedback;
