-- Requêtes de lecture du domaine présence partagées entre back-rex-admin
-- (page web Pointage) et back-rex-eleve (écrans mobiles prof/gestionnaire).
-- Les handlers HTTP et les règles d'autorisation restent propres à chaque
-- service ; seules les requêtes et leurs DTO de données sont communs.

-- name: GetSeance :one
SELECT s.id, s.matiere_id, s.code, s.opened_at, s.closed_at,
       s.late_after_minutes, s.prof_id, m.name AS matiere_name
FROM seance s JOIN matiere m ON m.id = s.matiere_id WHERE s.id = @id;

-- name: ListPresence :many
-- `justifie` est une COUCHE posée sur l'absence, pas un statut : le pointage
-- l'emporte toujours (un étudiant excusé qui a scanné reste PRESENT), les
-- écrans n'affichent « Excusé » que lorsque le statut dérivé vaut ABSENT.
-- Lecture par la VUE justification_active : une justification révoquée ne doit
-- jamais remonter sur une feuille de présence.
--
-- GROUP BY plutôt que le DISTINCT d'origine : un élève inscrit à plusieurs
-- groupes de la promotion produisait déjà des doublons, et deux justifications
-- actives qui se chevauchent en produiraient d'autres que DISTINCT ne saurait
-- pas fusionner (les lignes ne diffèrent que par j.id). bool_or les réduit sans
-- toucher au tri alphabétique — DISTINCT ON aurait imposé ORDER BY u.id.
SELECT u.id AS user_id, u.name, u.surname,
       COALESCE(p.statut, 'ABSENT')::text AS statut, p.pointe_at,
       bool_or(j.id IS NOT NULL) AS justifie
FROM seance s
JOIN matiere m       ON m.id = s.matiere_id
JOIN periode pe      ON pe.id = m.periode_id
JOIN promotion pr    ON pr.id = pe.promotion_id
JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
JOIN eleve_groupe eg ON eg.id_groupe = g.id
JOIN "user" u        ON u.id = eg.num_etudiant
LEFT JOIN pointage p ON p.user_id = u.id AND p.seance_id = s.id
LEFT JOIN justification_seance js ON js.seance_id = s.id
LEFT JOIN justification_active j  ON j.id = js.justification_id AND j.user_id = u.id
WHERE s.id = @seance_id
GROUP BY u.id, u.name, u.surname, p.statut, p.pointe_at
ORDER BY u.surname, u.name;

-- name: ListPresenceHorsGroupe :many
-- Un élève hors groupe a forcément pointé (la ligne vient de `pointage`) : son
-- statut n'est jamais ABSENT et `justifie` ne s'affichera donc jamais. La
-- colonne est portée malgré tout, pour que les quatre requêtes de présence
-- exposent le même contrat aux appelants.
SELECT u.id AS user_id, u.name, u.surname,
       p.statut::text AS statut, p.pointe_at,
       bool_or(j.id IS NOT NULL) AS justifie
FROM pointage p
JOIN "user" u ON u.id = p.user_id
LEFT JOIN justification_seance js ON js.seance_id = p.seance_id
LEFT JOIN justification_active j  ON j.id = js.justification_id AND j.user_id = u.id
WHERE p.seance_id = @seance_id
  AND u.id NOT IN (
    SELECT eg.num_etudiant
    FROM seance s
    JOIN matiere m       ON m.id = s.matiere_id
    JOIN periode pe      ON pe.id = m.periode_id
    JOIN promotion pr    ON pr.id = pe.promotion_id
    JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
    JOIN eleve_groupe eg ON eg.id_groupe = g.id
    WHERE s.id = @seance_id
  )
GROUP BY u.id, u.name, u.surname, p.statut, p.pointe_at
ORDER BY u.surname, u.name;

-- name: ListPresenceBySeances :many
-- Variante multi-séances de ListPresence, pour l'export d'un semestre entier :
-- une seule requête, regroupement par seance_id côté Go. Un appel par séance
-- ferait ici plus de cent allers-retours.
-- `justifie` suit la même règle qu'en unitaire : sans elle, la même séance
-- dirait « Excusé » sur sa feuille et « Absent » dans le PDF semestriel.
SELECT s.id AS seance_id, u.id AS user_id, u.name, u.surname,
       COALESCE(p.statut, 'ABSENT')::text AS statut, p.pointe_at,
       bool_or(j.id IS NOT NULL) AS justifie
FROM seance s
JOIN matiere m       ON m.id = s.matiere_id
JOIN periode pe      ON pe.id = m.periode_id
JOIN promotion pr    ON pr.id = pe.promotion_id
JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
JOIN eleve_groupe eg ON eg.id_groupe = g.id
JOIN "user" u        ON u.id = eg.num_etudiant
LEFT JOIN pointage p ON p.user_id = u.id AND p.seance_id = s.id
LEFT JOIN justification_seance js ON js.seance_id = s.id
LEFT JOIN justification_active j  ON j.id = js.justification_id AND j.user_id = u.id
WHERE s.id = ANY(@seance_ids::bigint[])
GROUP BY s.id, u.id, u.name, u.surname, p.statut, p.pointe_at
ORDER BY s.id, u.surname, u.name;

-- name: ListPresenceHorsGroupeBySeances :many
-- Pendant multi-séances de ListPresenceHorsGroupe. Le NOT IN de la version
-- unitaire devient un NOT EXISTS corrélé sur p.seance_id : le sous-ensemble
-- d'inscrits diffère d'une séance à l'autre (groupe_id propre à la séance).
SELECT p.seance_id, u.id AS user_id, u.name, u.surname,
       p.statut::text AS statut, p.pointe_at,
       bool_or(j.id IS NOT NULL) AS justifie
FROM pointage p
JOIN "user" u ON u.id = p.user_id
LEFT JOIN justification_seance js ON js.seance_id = p.seance_id
LEFT JOIN justification_active j  ON j.id = js.justification_id AND j.user_id = u.id
WHERE p.seance_id = ANY(@seance_ids::bigint[])
  AND NOT EXISTS (
    SELECT 1
    FROM seance s
    JOIN matiere m       ON m.id = s.matiere_id
    JOIN periode pe      ON pe.id = m.periode_id
    JOIN promotion pr    ON pr.id = pe.promotion_id
    JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
    JOIN eleve_groupe eg ON eg.id_groupe = g.id
    WHERE s.id = p.seance_id AND eg.num_etudiant = u.id
  )
GROUP BY p.seance_id, u.id, u.name, u.surname, p.statut, p.pointe_at
ORDER BY p.seance_id, u.surname, u.name;

-- name: ActivateSeance :one
-- Ouvre le pointage d'une séance déjà planifiée : lui assigne un code court.
-- Idempotence assurée par WHERE code IS NULL (0 ligne si déjà activée).
UPDATE seance
SET code = @code, opened_at = NOW(), late_after_minutes = @late_after_minutes
WHERE id = @id AND code IS NULL
RETURNING id, code, opened_at;

-- name: CloseSeance :exec
UPDATE seance SET closed_at = CURRENT_TIMESTAMP WHERE id = @id AND closed_at IS NULL;

-- name: ListSeancesJour :many
-- Séances (toutes) dont le début tombe dans la fenêtre [day_start, day_end).
-- La fenêtre est calculée côté Go pour maîtriser le fuseau horaire.
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(s.salle, '') AS salle,
       COALESCE(s.prof, '') AS prof,
       COALESCE(g.name, pr.name, '') AS promo,
       s.opened_at, s.closed_at,
       (s.code IS NOT NULL)::bool AS activated
FROM seance s
JOIN matiere m         ON m.id = s.matiere_id
LEFT JOIN groupe g     ON g.id = s.groupe_id
LEFT JOIN promotion pr ON pr.id = s.promotion_id
WHERE s.cancelled_at IS NULL
  AND s.starts_at >= @day_start AND s.starts_at < @day_end
ORDER BY s.starts_at;

-- name: ListSeancesJourByProf :many
-- Variante filtrée pour un prof : uniquement ses séances (prof_id = user.id).
-- Le filtrage se fait ici, côté SQL, jamais après coup dans le handler.
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(s.salle, '') AS salle,
       COALESCE(s.prof, '') AS prof,
       COALESCE(g.name, pr.name, '') AS promo,
       s.opened_at, s.closed_at,
       (s.code IS NOT NULL)::bool AS activated
FROM seance s
JOIN matiere m         ON m.id = s.matiere_id
LEFT JOIN groupe g     ON g.id = s.groupe_id
LEFT JOIN promotion pr ON pr.id = s.promotion_id
WHERE s.cancelled_at IS NULL
  AND s.prof_id = @prof_id
  AND s.starts_at >= @day_start AND s.starts_at < @day_end
ORDER BY s.starts_at;
