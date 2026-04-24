-- name: ListFeedbacks :many
SELECT id, content, created_at, promotion
FROM feedback f
LEFT JOIN student s ON f.user_id = s.user_id
ORDER BY f.created_at DESC;

-- name: ListFeedbacksWithClassification :many
SELECT f.id, f.content, f.created_at,
       c.promotion, c.groupe,
       c.categorie, c.sous_categorie, c.sentiment, c.urgence, c.resume,
       f.strongbox
FROM feedback f
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
ORDER BY f.created_at DESC;

-- name: ListPendingFeedbacks :many
SELECT f.id, f.content, f.user_id, f.season_id,
       s.promotion,
       string_agg(g.name, ', ') AS groupe
FROM feedback f
LEFT JOIN student s ON s.user_id = f.user_id
LEFT JOIN eleve_groupe eg ON eg.num_etudiant = f.user_id
LEFT JOIN groupe g ON g.id = eg.id_groupe
LEFT JOIN feedback_classification c ON c.feedback_id = f.id
WHERE c.feedback_id IS NULL
GROUP BY f.id, f.content, f.user_id, f.season_id, s.promotion
ORDER BY f.created_at ASC;

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