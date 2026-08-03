-- name: GetUnreadReponsesByPseudo :many
SELECT r.id, r.message_id, r.contenu, r.auteur, r.cree_le, r.lu
FROM reponse r
JOIN feedback f ON r.message_id = f.message_id
WHERE f.pseudo = $1 AND r.lu = false;

-- name: MarkReponsesByPseudoAsRead :exec
UPDATE reponse SET lu = true
WHERE message_id IN (
    SELECT message_id FROM feedback WHERE pseudo = $1
) AND lu = false;

-- name: GetChatHistoryByPseudo :many
-- Fil de discussion de l'étudiant : ses propres feedbacks ('me') et les
-- réponses des gestionnaires ('other'), sur une fenêtre glissante.
--
-- content est NULL tant que le message n'est pas publié : pour SON propre
-- message (authentifié par le pseudo), l'étudiant voit le contenu modéré une
-- fois publié, sinon son texte d'origine (raw_content). COALESCE évite aussi
-- l'échec de scan d'un NULL dans une string Go.
--
-- Cas REJECTED : raw_content a été remplacé par sa version chiffrée (age) au
-- moment du refus. On ne le renvoie donc JAMAIS — le blob age ne doit pas
-- sortir du serveur. Le motif du refus est fourni par /feedback/status.
SELECT
    f.message_id AS id,
    (CASE WHEN f.moderation_status = 'REJECTED'
          THEN sqlc.arg(rejected_placeholder)::text
          ELSE COALESCE(f.content, f.raw_content, '')
     END)::text AS text,
    'me'::text AS source,
    f.created_at AS ts
FROM feedback f
WHERE f.pseudo = sqlc.arg(pseudo) AND f.created_at >= sqlc.arg(since)
UNION ALL
SELECT
    'reponse-' || r.id::text AS id,
    r.contenu                AS text,
    'other'::text            AS source,
    r.cree_le                AS ts
FROM reponse r
JOIN feedback f ON r.message_id = f.message_id
WHERE f.pseudo = sqlc.arg(pseudo) AND r.cree_le >= sqlc.arg(since)
ORDER BY ts ASC;
