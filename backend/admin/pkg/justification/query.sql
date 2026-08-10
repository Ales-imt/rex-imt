-- Justifications d'absence (« excuses ») — saisie réservée au gestionnaire
-- d'année, exclusivement depuis le web admin. Aucune de ces requêtes n'écrit
-- dans `pointage` ni dans `presence_ledger` : l'excuse est une couche posée sur
-- l'absence, pas un fait de pointage.
--
-- Les trois tables sont append-only (trigger deny_mutation) : il n'y a donc ici
-- ni UPDATE ni DELETE. « Modifier » = révoquer + insérer une nouvelle version
-- portant replaces_id ; « annuler » = insérer une révocation.

-- name: ListSeancesCouvertes :many
-- Séances recouvertes par une plage, pour l'aperçu du dialogue de saisie et
-- pour la matérialisation de justification_seance.
--
-- La résolution de groupe reprend EXACTEMENT celle de ListPresence
-- (seance → matiere → periode → promotion → groupe → eleve_groupe) : toute
-- divergence permettrait de justifier une séance absente de la feuille de
-- présence. Le GROUP BY absorbe les élèves inscrits à plusieurs groupes.
--
-- `&&` (chevauchement) et non `<@` : un étudiant souffrant à partir de 10 h a
-- bien manqué le cours commencé à 9 h 45.
--
-- Les séances sans horaire complet sont écartées : tstzrange(NULL, x) est une
-- plage non bornée, qui chevaucherait n'importe quelle demande.
SELECT s.id, s.starts_at, s.ends_at,
       m.name AS matiere_name,
       COALESCE(s.salle, '') AS salle,
       COALESCE(p.statut, 'ABSENT')::text AS statut
FROM seance s
JOIN matiere m       ON m.id = s.matiere_id
JOIN periode pe      ON pe.id = m.periode_id
JOIN promotion pr    ON pr.id = pe.promotion_id
JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
JOIN eleve_groupe eg ON eg.id_groupe = g.id
LEFT JOIN pointage p ON p.seance_id = s.id AND p.user_id = @user_id
WHERE eg.num_etudiant = @user_id
  AND s.cancelled_at IS NULL
  AND s.starts_at IS NOT NULL
  AND s.ends_at IS NOT NULL
  AND tstzrange(s.starts_at, s.ends_at) && @periode::tstzrange
GROUP BY s.id, s.starts_at, s.ends_at, m.name, s.salle, p.statut
ORDER BY s.starts_at;

-- name: ListJustificationsChevauchantes :many
-- Justifications ACTIVES de l'étudiant chevauchant déjà la plage demandée.
-- Rien ne l'interdit en base (l'interdire casserait les corrections), mais le
-- dialogue doit le signaler : deux excuses concurrentes sur la même séance sont
-- fusionnées par le bool_or de ListPresence, et annuler l'une n'a alors aucun
-- effet visible — un bug invisible et pénible à diagnostiquer.
SELECT j.id, lower(j.periode)::timestamptz AS debut, upper(j.periode)::timestamptz AS fin
FROM justification_active j
WHERE j.user_id = @user_id
  AND j.periode && @periode::tstzrange
  AND (sqlc.narg('exclude_id')::bigint IS NULL OR j.id <> sqlc.narg('exclude_id')::bigint)
ORDER BY lower(j.periode);

-- name: CreateJustification :one
INSERT INTO justification (user_id, periode, replaces_id, created_by)
VALUES (@user_id, @periode::tstzrange, @replaces_id, @created_by)
RETURNING id, created_at;

-- name: InsertJustificationSeances :exec
-- DISTINCT : la liste vient d'un aperçu potentiellement redondant, et la table
-- n'accepte ni ON CONFLICT DO UPDATE ni correction a posteriori.
INSERT INTO justification_seance (justification_id, seance_id)
SELECT DISTINCT @justification_id::bigint, s.id
FROM seance s
WHERE s.id = ANY(@seance_ids::bigint[])
ON CONFLICT DO NOTHING;

-- name: RevokeJustification :exec
-- Annulation ET première étape d'une correction. Aucune ligne n'est modifiée :
-- c'est une écriture inverse, au sens comptable.
INSERT INTO justification_revocation (justification_id, revoked_by)
VALUES (@justification_id, @revoked_by);

-- name: GetJustification :one
-- Le statut dérivé est calculé en SQL, jamais après coup en Go : une chaîne de
-- corrections doit se lire de la même façon ici et dans la liste.
SELECT j.id, j.user_id, u.name, u.surname,
       lower(j.periode)::timestamptz AS debut,
       upper(j.periode)::timestamptz AS fin,
       j.replaces_id,
       nxt.id AS replaced_by_id,
       j.created_at, j.created_by,
       COALESCE(cb.name, '') AS created_by_name,
       COALESCE(cb.surname, '') AS created_by_surname,
       rev.revoked_at,
       (SELECT COUNT(*) FROM justification_seance js
          JOIN seance s ON s.id = js.seance_id
         WHERE js.justification_id = j.id AND s.cancelled_at IS NULL)::bigint AS nb_seances,
       (CASE WHEN nxt.id IS NOT NULL                THEN 'REMPLACEE'
             WHEN rev.justification_id IS NOT NULL  THEN 'ANNULEE'
             ELSE 'ACTIVE' END)::text AS statut
FROM justification j
JOIN "user" u                          ON u.id = j.user_id
LEFT JOIN "user" cb                    ON cb.id = j.created_by
LEFT JOIN justification_revocation rev ON rev.justification_id = j.id
LEFT JOIN justification nxt            ON nxt.replaces_id = j.id
WHERE j.id = @id;

-- name: ListJustifications :many
-- Liste de l'écran de gestion. Par défaut include_revoked = false : une
-- révocation existe aussi bien pour une annulation que pour une correction,
-- le filtre laisse donc exactement les justifications ACTIVE.
SELECT j.id, j.user_id, u.name, u.surname,
       lower(j.periode)::timestamptz AS debut,
       upper(j.periode)::timestamptz AS fin,
       j.replaces_id,
       nxt.id AS replaced_by_id,
       j.created_at, j.created_by,
       COALESCE(cb.name, '') AS created_by_name,
       COALESCE(cb.surname, '') AS created_by_surname,
       rev.revoked_at,
       (SELECT COUNT(*) FROM justification_seance js
          JOIN seance s ON s.id = js.seance_id
         WHERE js.justification_id = j.id AND s.cancelled_at IS NULL)::bigint AS nb_seances,
       (CASE WHEN nxt.id IS NOT NULL               THEN 'REMPLACEE'
             WHEN rev.justification_id IS NOT NULL THEN 'ANNULEE'
             ELSE 'ACTIVE' END)::text AS statut
FROM justification j
JOIN "user" u                          ON u.id = j.user_id
LEFT JOIN "user" cb                    ON cb.id = j.created_by
LEFT JOIN justification_revocation rev ON rev.justification_id = j.id
LEFT JOIN justification nxt            ON nxt.replaces_id = j.id
WHERE (sqlc.narg('promo_id')::bigint IS NULL OR EXISTS (
         SELECT 1 FROM eleve_groupe eg
         JOIN groupe g ON g.id = eg.id_groupe
         WHERE eg.num_etudiant = j.user_id AND g.promo_id = sqlc.narg('promo_id')::bigint))
  AND (sqlc.narg('periode_id')::bigint IS NULL OR EXISTS (
         SELECT 1 FROM justification_seance js
         JOIN seance s  ON s.id = js.seance_id
         JOIN matiere m ON m.id = s.matiere_id
         WHERE js.justification_id = j.id AND m.periode_id = sqlc.narg('periode_id')::bigint))
  AND (sqlc.narg('user_id')::integer IS NULL OR j.user_id = sqlc.narg('user_id')::integer)
  AND (sqlc.narg('recherche')::text IS NULL
       OR u.name ILIKE '%' || sqlc.narg('recherche')::text || '%'
       OR u.surname ILIKE '%' || sqlc.narg('recherche')::text || '%')
  AND (@include_revoked::boolean OR rev.justification_id IS NULL)
ORDER BY lower(j.periode) DESC, j.id DESC;

-- name: ListJustificationSeances :many
-- Détail d'une justification (dépliage de ligne). Les séances annulées après
-- coup sont écartées : la ligne justification_seance subsiste — rien n'est
-- jamais supprimé — mais annoncer une séance qui n'existe plus serait faux.
-- Même filtre que le compteur nb_seances, sous peine de deux comptes différents.
SELECT s.id, s.starts_at, s.ends_at,
       m.name AS matiere_name,
       COALESCE(s.salle, '') AS salle,
       COALESCE(p.statut, 'ABSENT')::text AS statut
FROM justification_seance js
JOIN justification j ON j.id = js.justification_id
JOIN seance s        ON s.id = js.seance_id
JOIN matiere m       ON m.id = s.matiere_id
LEFT JOIN pointage p ON p.seance_id = s.id AND p.user_id = j.user_id
WHERE js.justification_id = @justification_id
  AND s.cancelled_at IS NULL
ORDER BY s.starts_at;
