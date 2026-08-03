-- name: ListPendingModeration :many
-- Feedbacks en attente de modération. N'expose que le texte brut à modérer et
-- sa promotion (qui sert au filtrage de la file) : ni strongbox, ni pseudo
-- (choisi librement, peut contenir un nom), ni aucune donnée désignant un
-- auteur en particulier — une promotion regroupe une cohorte entière.
SELECT id, raw_content, created_at, promotion
FROM feedback
WHERE moderation_status = 'PENDING'
ORDER BY created_at ASC;

-- name: ApproveFeedback :execrows
-- Publication : pose le contenu modéré, passe à PUBLISHED, trace le modérateur
-- et efface raw_content — le tout dans un UNIQUE UPDATE pour que le trigger de
-- notification IA ne se déclenche qu'une fois. Si content_modere est vide (pas
-- d'édition), on republie l'ancien raw_content (évalué sur l'ancienne ligne).
UPDATE feedback
SET content           = COALESCE(NULLIF($2::text, ''), raw_content),
    moderation_status = 'PUBLISHED',
    moderated_by      = $3,
    moderated_at      = now(),
    raw_content       = NULL
WHERE id = $1
  AND moderation_status = 'PENDING';

-- name: GetPendingRawContent :one
-- Récupère le texte brut d'un feedback encore en attente, pour le chiffrer au
-- moment du refus (le clair est ensuite remplacé par sa version chiffrée).
SELECT COALESCE(raw_content, '')::text AS raw_content
FROM feedback
WHERE id = $1
  AND moderation_status = 'PENDING';

-- name: RejectFeedback :execrows
-- Refus : passe à REJECTED avec motif et remplace raw_content par sa version
-- chiffrée (age) — le contenu n'est plus lisible en clair au repos. Il est
-- purgé ultérieurement (cf. rgpd).
UPDATE feedback
SET moderation_status = 'REJECTED',
    rejection_reason  = $2,
    moderated_by      = $3,
    raw_content       = $4,
    moderated_at      = now()
WHERE id = $1
  AND moderation_status = 'PENDING';


-- ===================== Verbatims d'évaluation =====================
-- Même pipeline que les feedbacks ci-dessus, appliqué à eval_verbatim :
-- raw_texte (brut, à modérer) → texte (publié) ou chiffrement age au refus.
-- Différence : la clé est un uuid, et la dimension (SUPPORTS / AMBIANCE / NPS)
-- est exposée au modérateur pour lui donner le contexte de la question posée.

-- name: ListPendingVerbatims :many
-- Verbatims en attente de modération. N'expose que le texte brut, sa dimension
-- et la promotion du cours évalué (pour le filtrage de la file) : ni strongbox,
-- ni session_id, qui relierait le verbatim à son auteur via eval_session.
--
-- La promotion se déduit du cours évalué (matiere → periode → promotion), pas
-- de l'auteur : la jointure ne touche jamais eval_session.pseudo.
SELECT ev.id, ev.raw_texte, ev.dimension, ev.created_at,
       COALESCE(pr.name, '')::text AS promotion
FROM eval_verbatim ev
JOIN eval_session es ON es.id = ev.session_id
JOIN matiere    m  ON m.id  = es.matiere_id
LEFT JOIN periode   pe ON pe.id = m.periode_id
LEFT JOIN promotion pr ON pr.id = pe.promotion_id
WHERE ev.moderation_status = 'PENDING'
ORDER BY ev.created_at ASC;

-- name: ApproveVerbatim :execrows
-- Publication : pose le texte modéré, passe à PUBLISHED, trace le modérateur
-- et efface raw_texte. Si texte_modere est vide (pas d'édition), on republie
-- l'ancien raw_texte (évalué sur l'ancienne ligne).
UPDATE eval_verbatim
SET texte             = COALESCE(NULLIF($2::text, ''), raw_texte),
    moderation_status = 'PUBLISHED',
    moderated_by      = $3,
    moderated_at      = now(),
    raw_texte         = NULL
WHERE id = $1
  AND moderation_status = 'PENDING';

-- name: GetPendingRawVerbatim :one
-- Récupère le texte brut d'un verbatim encore en attente, pour le chiffrer au
-- moment du refus (le clair est ensuite remplacé par sa version chiffrée).
SELECT COALESCE(raw_texte, '')::text AS raw_texte
FROM eval_verbatim
WHERE id = $1
  AND moderation_status = 'PENDING';

-- name: RejectVerbatim :execrows
-- Refus : passe à REJECTED avec motif et remplace raw_texte par sa version
-- chiffrée (age). Le verbatim n'est alors ni affiché, ni envoyé à l'IA ; il
-- est purgé ultérieurement (cf. rgpd).
UPDATE eval_verbatim
SET moderation_status = 'REJECTED',
    rejection_reason  = $2,
    moderated_by      = $3,
    raw_texte         = $4,
    moderated_at      = now()
WHERE id = $1
  AND moderation_status = 'PENDING';
