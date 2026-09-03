-- Lecture du planning depuis public.seance, alimentée toutes les 2 h par la
-- synchronisation de back-rex-admin (pkg/migration/planning.go). Source
-- nominale du programme : plus aucun appel au système amont à la volée.
--
-- Les trois requêtes filtrent sur [debut, fin) et écartent les séances
-- annulées (cancelled_at IS NOT NULL) : une séance supprimée en amont ne doit
-- pas rester affichée comme un cours fantôme.

-- name: ListProgrammeEleve :many
-- Le planning d'un élève est exactement l'ensemble des séances dont il est
-- attendu : il lit la vue seance_effectif_resolu, la seule écriture de cette
-- règle dans le dépôt (cf. presencedata.ListPresence). Un cours affiché ici
-- est donc toujours un cours où il peut être pointé, et réciproquement.
--
-- Une version antérieure recopiait la chaîne matière → période → promotion.
-- Elle perdait toutes les séances d'un cours mutualisé ciblant un groupe d'une
-- autre promotion que celle de la matière (journées de rentrée, remises à
-- niveau…), et les promettait pourtant identiques à la feuille de présence.
--
-- Conséquence assumée de la vue : pour une séance clôturée, c'est l'effectif
-- FIGÉ qui fait foi. Un élève ayant changé de groupe garde donc au planning
-- les séances passées où il était convoqué, comme sur la feuille.
--
-- `promo` est le nom de la PROMOTION et non celui du groupe : le filtre promo du
-- front mobile en dépend. Elle vient de seance.promotion_id, avec repli sur la
-- période de la matière pour les séances qui n'en portent pas — mêmes règles
-- que ListProgrammeProf. `groupe` est le nom du groupe (TD/TP) de la séance,
-- vide pour une séance de promotion entière.
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(sa.name, '') AS salle,
       COALESCE(s.prof, '')  AS prof,
       COALESCE(prs.name, prp.name, '') AS promo,
       COALESCE(g.name, '') AS groupe,
       COALESCE(s.remarque, '') AS remarque
FROM seance s
JOIN matiere m          ON m.id = s.matiere_id
JOIN seance_effectif_resolu er ON er.seance_id = s.id AND er.user_id = @user_id
LEFT JOIN salle sa      ON sa.id = s.salle_id
LEFT JOIN promotion prs ON prs.id = s.promotion_id
LEFT JOIN periode pe    ON pe.id = m.periode_id
LEFT JOIN promotion prp ON prp.id = pe.promotion_id
LEFT JOIN groupe g      ON g.id = s.groupe_id
WHERE s.cancelled_at IS NULL
  AND s.starts_at >= @debut AND s.starts_at < @fin
ORDER BY s.starts_at;

-- name: ListProgrammeProf :many
-- prof_id est résolu par la synchronisation depuis migration.prof_map. Une
-- séance dont le PRCLE amont est inconnu n'a pas de prof_id et n'apparaît donc
-- dans le planning de personne : c'est le comportement voulu, mieux vaut un
-- créneau manquant qu'un créneau attribué au mauvais enseignant.
--
-- La promo vient de seance.promotion_id, avec repli sur la période de la
-- matière pour les séances importées sans promotion. Le groupe, lui, vient
-- directement de s.groupe_id — pas d'ambiguïté à résoudre ici, prof et
-- gestionnaire ne voient chacun qu'une ligne par séance.
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(sa.name, '') AS salle,
       COALESCE(s.prof, '')  AS prof,
       COALESCE(prs.name, prp.name, '') AS promo,
       COALESCE(g.name, '') AS groupe,
       COALESCE(s.remarque, '') AS remarque
FROM seance s
JOIN matiere m          ON m.id = s.matiere_id
LEFT JOIN salle sa      ON sa.id = s.salle_id
LEFT JOIN promotion prs ON prs.id = s.promotion_id
LEFT JOIN periode pe    ON pe.id = m.periode_id
LEFT JOIN promotion prp ON prp.id = pe.promotion_id
LEFT JOIN groupe g      ON g.id = s.groupe_id
WHERE s.prof_id = @prof_id
  AND s.cancelled_at IS NULL
  AND s.starts_at >= @debut AND s.starts_at < @fin
ORDER BY s.starts_at;

-- name: ListProgrammeToutes :many
-- Gestionnaire : toutes les séances de la plage, sans filtrage par utilisateur,
-- comme presencedata.ListSeancesJour pour la journée courante.
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(sa.name, '') AS salle,
       COALESCE(s.prof, '')  AS prof,
       COALESCE(prs.name, prp.name, '') AS promo,
       COALESCE(g.name, '') AS groupe,
       COALESCE(s.remarque, '') AS remarque
FROM seance s
JOIN matiere m          ON m.id = s.matiere_id
LEFT JOIN salle sa      ON sa.id = s.salle_id
LEFT JOIN promotion prs ON prs.id = s.promotion_id
LEFT JOIN periode pe    ON pe.id = m.periode_id
LEFT JOIN promotion prp ON prp.id = pe.promotion_id
LEFT JOIN groupe g      ON g.id = s.groupe_id
WHERE s.cancelled_at IS NULL
  AND s.starts_at >= @debut AND s.starts_at < @fin
ORDER BY s.starts_at;
