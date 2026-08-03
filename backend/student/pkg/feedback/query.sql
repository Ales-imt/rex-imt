-- name: GetUserNameSurname :one
SELECT name, surname
FROM "user"
WHERE id = $1;

-- name: GetStudentPromoAndGroupes :one
SELECT
    p.name                                     AS promotion,
    string_agg(g.name, ', ' ORDER BY g.name)::text  AS groupes
FROM eleve_groupe eg
JOIN groupe    g ON eg.id_groupe   = g.id
JOIN promotion p ON g.promo_id     = p.id
WHERE eg.num_etudiant = $1
GROUP BY p.name
LIMIT 1;

-- name: InsertFeedbacks :one
-- Le texte est écrit dans raw_content (pour la modération uniquement) ;
-- content reste NULL tant que le message n'est pas publié (moderation_status
-- vaut PENDING par défaut).
INSERT INTO feedback (raw_content, created_at, strongbox, pseudo, message_id, promotion, groupe)
			VALUES ($1, $2, $3, $4, $5, $6, $7) returning id;

-- name: AnonymizeFeedback :exec
-- Auto-suppression à la demande de l'auteur : on efface aussi raw_content pour
-- ne laisser aucune trace du texte brut d'origine.
UPDATE feedback
SET
  content     = '[contenu supprimé à la demande de l''auteur]',
  raw_content = NULL,
  pseudo      = NULL,
  promotion   = NULL,
  groupe      = NULL
WHERE message_id = $1
  AND pseudo = $2;

-- name: GetFeedbackStatuses :many
-- Consultation par l'étudiant du statut de SES messages, à partir des
-- message_id qu'il détient et de son pseudo. Aucun lien auteur→feedback n'est
-- reconstruit côté serveur : la preuve de possession vient du couple
-- (message_id, pseudo) fourni par le client.
SELECT message_id, moderation_status, rejection_reason
FROM feedback
WHERE pseudo = $1
  AND message_id = ANY($2::text[]);

-- name: AnonymizePostitByFeedback :exec
UPDATE postit SET message_modere = '[contenu supprimé à la demande de l''auteur]'
WHERE message_id = $1;