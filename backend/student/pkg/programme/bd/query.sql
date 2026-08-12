-- Lecture du planning depuis public.seance, alimentée toutes les 2 h par la
-- synchronisation de back-rex-admin (pkg/migration/planning.go). Source
-- nominale du programme : plus aucun appel au système amont à la volée.
--
-- L'effectif est ici résolu sur eleve_groupe, donc sur l'état COURANT des
-- groupes : un élève ayant changé de groupe voit son planning passé recalculé.
-- C'est sans conséquence pour un planning, et inacceptable sur une feuille de
-- présence — d'où seance_effectif_resolu, que ces requêtes n'utilisent
-- volontairement PAS.
--
-- Les trois requêtes filtrent sur [debut, fin) et écartent les séances
-- annulées (cancelled_at IS NOT NULL) : une séance supprimée en amont ne doit
-- pas rester affichée comme un cours fantôme.

-- name: ListProgrammeEleve :many
-- Jointures identiques à presencedata.ListPresence : le planning d'un élève doit
-- désigner exactement les séances dont il est attendu, sans quoi il pourrait
-- être marqué absent d'un cours qui ne lui a jamais été annoncé.
--
-- `promo` est le nom de la PROMOTION et non celui du groupe : le filtre promo du
-- front mobile en dépend.
--
-- GROUP BY : un élève inscrit à plusieurs groupes d'une même promotion
-- produirait autant de copies de chaque séance de promo (groupe_id IS NULL).
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(s.salle, '') AS salle,
       COALESCE(s.prof, '')  AS prof,
       COALESCE(pr.name, '') AS promo
FROM seance s
JOIN matiere m       ON m.id = s.matiere_id
JOIN periode pe      ON pe.id = m.periode_id
JOIN promotion pr    ON pr.id = pe.promotion_id
JOIN groupe g        ON g.promo_id = pr.id AND (s.groupe_id IS NULL OR g.id = s.groupe_id)
JOIN eleve_groupe eg ON eg.id_groupe = g.id
WHERE eg.num_etudiant = @user_id
  AND s.cancelled_at IS NULL
  AND s.starts_at >= @debut AND s.starts_at < @fin
GROUP BY s.id, m.name, s.starts_at, s.ends_at, s.salle, s.prof, pr.name
ORDER BY s.starts_at;

-- name: ListProgrammeProf :many
-- prof_id est résolu par la synchronisation depuis migration.prof_map. Une
-- séance dont le PRCLE amont est inconnu n'a pas de prof_id et n'apparaît donc
-- dans le planning de personne : c'est le comportement voulu, mieux vaut un
-- créneau manquant qu'un créneau attribué au mauvais enseignant.
--
-- La promo vient de seance.promotion_id, avec repli sur la période de la
-- matière pour les séances importées sans promotion.
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(s.salle, '') AS salle,
       COALESCE(s.prof, '')  AS prof,
       COALESCE(prs.name, prp.name, '') AS promo
FROM seance s
JOIN matiere m          ON m.id = s.matiere_id
LEFT JOIN promotion prs ON prs.id = s.promotion_id
LEFT JOIN periode pe    ON pe.id = m.periode_id
LEFT JOIN promotion prp ON prp.id = pe.promotion_id
WHERE s.prof_id = @prof_id
  AND s.cancelled_at IS NULL
  AND s.starts_at >= @debut AND s.starts_at < @fin
ORDER BY s.starts_at;

-- name: ListProgrammeToutes :many
-- Gestionnaire : toutes les séances de la plage, sans filtrage par utilisateur,
-- comme presencedata.ListSeancesJour pour la journée courante.
SELECT s.id, s.matiere_id, m.name AS matiere_name,
       s.starts_at, s.ends_at,
       COALESCE(s.salle, '') AS salle,
       COALESCE(s.prof, '')  AS prof,
       COALESCE(prs.name, prp.name, '') AS promo
FROM seance s
JOIN matiere m          ON m.id = s.matiere_id
LEFT JOIN promotion prs ON prs.id = s.promotion_id
LEFT JOIN periode pe    ON pe.id = m.periode_id
LEFT JOIN promotion prp ON prp.id = pe.promotion_id
WHERE s.cancelled_at IS NULL
  AND s.starts_at >= @debut AND s.starts_at < @fin
ORDER BY s.starts_at;
