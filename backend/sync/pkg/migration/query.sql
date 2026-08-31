-- name: Now :one
-- Horloge de la BASE, prise en début de transaction d'écriture. Elle sert de
-- borne à ListSeancesWebdfdNonVues, dont les lignes portent un last_seen_at
-- écrit par now() : comparer les deux depuis la même horloge supprime la dérive
-- entre l'application et le serveur. Dans une transaction, now() vaut
-- transaction_timestamp(), constant du BEGIN au COMMIT.
SELECT now()::timestamptz;

-- name: GetAnneeByDate :one
-- La date est castée avant comparaison : debut et fin sont des `date`, et un
-- timestamptz non casté les promeut à minuit — l'année cesserait alors d'être
-- trouvée dès 00:00:01 le dernier jour de la période.
SELECT id, debut, fin FROM public.annee WHERE @jour::date BETWEEN debut AND fin;

-- name: ListAnnees :many
SELECT id, debut, fin FROM public.annee ORDER BY debut;

-- name: CreateInconnuPromotion :exec
INSERT INTO public.promotion (id, name) VALUES (0, 'inconnu') ON CONFLICT (id) DO NOTHING;

-- name: CreatePromotion :one
INSERT INTO public.promotion (name) VALUES ($1) RETURNING id;

-- name: UpdatePromotionName :exec
UPDATE public.promotion SET name = $1 WHERE id = $2;

-- name: UpdatePromotionAnnee :exec
UPDATE public.promotion SET annee_id = $1 WHERE id = $2;

-- name: GetPromotionBySource :one
SELECT internal_id FROM migration.promotion_map WHERE source = $1 AND external_id = $2 AND annee = $3;

-- name: UpsertPromotionMap :exec
INSERT INTO migration.promotion_map (internal_id, source, external_id, annee, last_seen_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (source, external_id, annee) DO UPDATE
  SET internal_id  = EXCLUDED.internal_id,
      last_seen_at = now();

-- name: ListPromotionWebdfdIDs :many
SELECT internal_id, external_id FROM migration.promotion_map WHERE source = 'webdfd' AND annee = $1;

-- name: GetMatiereBySource :one
SELECT internal_id FROM migration.matiere_map WHERE source = $1 AND external_id = $2 AND annee = $3;

-- name: UpsertMatiereMap :exec
INSERT INTO migration.matiere_map (internal_id, source, external_id, annee, last_seen_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (source, external_id, annee) DO UPDATE
  SET internal_id  = EXCLUDED.internal_id,
      last_seen_at = now();

-- name: CreateMatiere :one
INSERT INTO public.matiere (name, annee) VALUES ($1, $2) RETURNING id;

-- name: UpdateMatiereName :exec
UPDATE public.matiere SET name = $1 WHERE id = $2;

-- name: UpdateMatiereAnnee :exec
UPDATE public.matiere SET annee = $1 WHERE id = $2;

-- name: UpsertPeriode :one
INSERT INTO public.periode (name, promotion_id, annee)
VALUES ($1, $2, $3)
ON CONFLICT (name, promotion_id, annee) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: UpdateMatierePeriode :exec
UPDATE public.matiere SET periode_id = $1 WHERE id = $2;

-- name: GetGroupeBySource :one
SELECT internal_id FROM migration.groupe_map WHERE source = $1 AND external_id = $2 AND annee = $3;

-- name: UpsertGroupeMap :exec
INSERT INTO migration.groupe_map (internal_id, source, external_id, annee, last_seen_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (source, external_id, annee) DO UPDATE
  SET internal_id  = EXCLUDED.internal_id,
      last_seen_at = now();

-- name: CreateGroupe :one
INSERT INTO public.groupe (name, taille, promo_id) VALUES ($1, 0, $2) RETURNING id;

-- name: UpdateGroupeName :exec
UPDATE public.groupe SET name = $1 WHERE id = $2;

-- name: UpdateGroupeTaille :exec
UPDATE public.groupe SET taille = $1 WHERE id = $2;

-- name: ListGroupeWebdfdIDs :many
SELECT internal_id, external_id FROM migration.groupe_map WHERE source = 'webdfd' AND annee = $1;

-- name: GetGroupeLabel :one
SELECT COALESCE(name, '') AS name FROM public.groupe WHERE id = $1;

-- name: GetProfBySource :one
SELECT internal_id FROM migration.prof_map WHERE source = $1 AND external_id = $2;

-- name: DeleteStaleProfMap :exec
DELETE FROM migration.prof_map WHERE internal_id = $1 AND source = $2 AND external_id != $3;

-- name: UpsertProfMap :exec
INSERT INTO migration.prof_map (internal_id, source, external_id, last_seen_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (source, external_id) DO UPDATE
  SET internal_id  = EXCLUDED.internal_id,
      last_seen_at = now();

-- name: CreateUserProf :one
INSERT INTO public."user" (email, name, surname, roles) VALUES ($1, $2, $3, ARRAY['PROF']) RETURNING id;

-- name: AddProfRole :exec
UPDATE public."user" SET roles = array_append(roles, 'PROF')
WHERE id = $1 AND NOT ('PROF' = ANY(roles));

-- name: GetUserBySource :one
SELECT internal_id FROM migration.user_map WHERE source = $1 AND external_id = $2;

-- name: GetUserByEmail :one
SELECT id FROM public."user" WHERE email = $1;

-- name: InsertUserEleve :one
INSERT INTO public."user" (email, name, surname, roles) VALUES ($1, $2, $3, ARRAY['ELEVE']) RETURNING id;

-- name: InsertStudent :exec
-- ON CONFLICT : le rattachement par email (SyncEleves étape 2) l'appelle sur un
-- compte qui peut déjà porter sa ligne student.
INSERT INTO public.student (user_id) VALUES ($1) ON CONFLICT DO NOTHING;

-- name: AddEleveRole :exec
-- Pendant de AddProfRole. Un compte retrouvé par email lors de la
-- synchronisation des élèves doit porter le rôle, sans quoi les deux chemins
-- d'arrivée (création, rattachement) produisent des comptes différents.
UPDATE public."user" SET roles = array_append(roles, 'ELEVE')
WHERE id = $1 AND NOT ('ELEVE' = ANY(roles));

-- name: UpdateUserNames :exec
UPDATE public."user" SET name = $1, surname = $2 WHERE id = $3;

-- name: DeleteStaleUserMap :exec
DELETE FROM migration.user_map WHERE internal_id = $1 AND source = $2 AND external_id != $3;

-- name: UpsertUserMap :exec
INSERT INTO migration.user_map (internal_id, source, external_id, last_seen_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (source, external_id) DO UPDATE
  SET internal_id  = EXCLUDED.internal_id,
      last_seen_at = now();

-- name: GetSalleBySource :one
SELECT internal_id FROM migration.salle_map WHERE source = $1 AND external_id = $2;

-- name: CreateSalle :one
INSERT INTO public.salle (name, capacite, type) VALUES ($1, $2, $3) RETURNING id;

-- name: UpdateSalle :exec
UPDATE public.salle SET name = $1, capacite = $2, type = $3 WHERE id = $4;

-- name: UpsertSalleMap :exec
INSERT INTO migration.salle_map (internal_id, source, external_id, last_seen_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (source, external_id) DO UPDATE
  SET internal_id  = EXCLUDED.internal_id,
      last_seen_at = now();

-- name: ListSallesParSource :many
-- Index SACLE → salle du cycle. JOIN et non LEFT JOIN : une salle sans
-- correspondance amont n'est rattachable par rien — le nom ne sert jamais de
-- clé — et n'a donc pas à figurer dans le résolveur.
SELECT sa.id, sa.name, m.external_id
FROM public.salle sa
JOIN migration.salle_map m ON m.internal_id = sa.id AND m.source = $1
ORDER BY sa.id;

-- name: GetSeanceBySource :one
SELECT internal_id FROM migration.seance_map WHERE source = $1 AND external_id = $2;

-- name: CreateSeance :one
INSERT INTO public.seance
  (matiere_id, starts_at, ends_at, salle, salle_id, prof, promotion_id, groupe_id, prof_id, remarque, opened_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $2)
RETURNING id;

-- name: AttacherJustificationsSeance :exec
-- Rattrape la couverture des excuses pour une séance qui vient d'être créée par
-- la synchronisation du planning.
--
-- justification_seance est matérialisée à la saisie de l'excuse : une séance
-- apparue APRÈS, dans une plage déjà couverte, ne serait sinon jamais excusée,
-- et silencieusement. L'effectif attendu vient de seance_effectif_resolu, comme
-- pour ListSeancesCouvertes et ListPresence — deux règles divergentes
-- produiraient deux couvertures différentes selon le chemin d'arrivée.
--
-- DISTINCT conservé : la vue est unique par couple, mais un même élève peut
-- porter plusieurs justifications actives chevauchant la séance.
-- Seules les justifications ACTIVES sont rattachées.
INSERT INTO public.justification_seance (justification_id, seance_id)
SELECT DISTINCT j.id, s.id
FROM public.seance s
JOIN public.seance_effectif_resolu er ON er.seance_id = s.id
JOIN public.justification_active j    ON j.user_id = er.user_id
WHERE s.id = @seance_id
  AND s.cancelled_at IS NULL
  AND s.starts_at IS NOT NULL
  AND s.ends_at IS NOT NULL
  AND tstzrange(s.starts_at, s.ends_at) && j.periode
ON CONFLICT DO NOTHING;

-- name: UpdateSeance :one
-- cancelled_at = NULL : une séance annulée par MarkSeancesAnnulees puis
-- rétablie dans le planning amont doit redevenir visible. Sans cette remise à
-- zéro, une erreur de saisie corrigée en amont laisserait le cours invisible
-- pour toujours.
--
-- prof_id et groupe_id ne sont PAS écrasés aveuglément. L'amont distingue deux
-- situations que l'UPDATE confondait : le créneau ne porte aucun PRCLE/GRCLE
-- (l'amont dit « pas de prof », « pas de groupe » → NULL), ou il en porte un
-- que la synchronisation n'a pas su résoudre (prof absent de prof_map faute
-- d'email, groupe encore inconnu → on garde la valeur en place). L'enjeu est
-- surtout sur groupe_id : à NULL, seance_effectif_resolu élargit
-- silencieusement l'effectif attendu à la promotion entière.
--
-- Le CTE `cible` capte l'état précédent et calcule les valeurs retenues :
-- RETURNING ne rend que les valeurs nouvelles, alors que l'appelant a besoin de
-- savoir si la séance était annulée (compteur de résurrections) et si son
-- rattachement aux justifications doit être revu.
WITH avant AS (
  SELECT sa.id, sa.cancelled_at, sa.starts_at, sa.ends_at, sa.matiere_id, sa.groupe_id, sa.prof_id
  FROM public.seance sa WHERE sa.id = @seance_id
)
UPDATE public.seance s
-- salle et salle_id viennent du MÊME appel au résolveur (cf. salles.go) : le
-- texte est le name de la salle résolue par le SACLE, et ne retombe sur le
-- libellé du créneau que lorsque la clé ne résout pas. salle_id est écrasé sans
-- la précaution de groupe_id/prof_id : le résolveur est reconstruit depuis la
-- base à chaque cycle, un NULL signifie bien « SACLE absent ou inconnu du
-- référentiel », et non « on n'a pas su résoudre faute de données ».
SET matiere_id   = @matiere_id,
    starts_at    = @starts_at,
    ends_at      = @ends_at,
    salle        = @salle,
    salle_id     = @salle_id,
    prof         = @prof,
    promotion_id = @promotion_id,
    groupe_id    = COALESCE(@groupe_id, CASE WHEN @grcle_vide::bool THEN NULL ELSE avant.groupe_id END),
    prof_id      = COALESCE(@prof_id,   CASE WHEN @prcle_vide::bool THEN NULL ELSE avant.prof_id   END),
    remarque     = @remarque,
    cancelled_at = NULL
FROM avant
WHERE s.id = avant.id
RETURNING
  (avant.cancelled_at IS NOT NULL)::bool AS ressuscitee,
  -- rattacher : la couverture des excuses doit être recalculée. Vrai quand la
  -- séance redevient visible, quand sa plage horaire bouge, ou quand son
  -- effectif attendu change (matière → période → promotion, ou groupe).
  --
  -- Le groupe est comparé à la valeur RETENUE, en répétant le COALESCE du SET
  -- plutôt qu'en testant le paramètre. Ce n'est pas un choix de style : un
  -- `@groupe_id IS NOT NULL` n'apporte aucune information de type, et
  -- PostgreSQL analyse le RETURNING AVANT la liste SET — le paramètre y serait
  -- donc encore non typé, et la requête entière échouerait au PREPARE avec
  -- « could not determine data type of parameter » (SQLSTATE 42P08).
  (avant.cancelled_at IS NOT NULL
   OR avant.starts_at  IS DISTINCT FROM @starts_at
   OR avant.ends_at    IS DISTINCT FROM @ends_at
   OR avant.matiere_id IS DISTINCT FROM @matiere_id
   OR avant.groupe_id  IS DISTINCT FROM
        COALESCE(@groupe_id, CASE WHEN @grcle_vide::bool THEN NULL ELSE avant.groupe_id END)
  )::bool AS rattacher;

-- name: ListSeancesWebdfdNonVues :many
-- Séances webdfd dont la correspondance n'a pas été rafraîchie pendant le cycle
-- courant (UpsertSeanceMap remet last_seen_at à now() à chaque passage) : ce
-- sont les CANDIDATES à l'annulation, pas les élues.
--
-- La décision finale se prend en Go (seancesPerimees) parce qu'elle dépend
-- d'informations que la base ignore : quelles promos ont réellement été
-- récupérées pendant ce cycle, et sur quelle plage de dates. Une promo dont le
-- fetch a échoué verrait sinon tout son planning annulé par un simple incident
-- réseau.
SELECT sm.internal_id, sm.external_id, s.promotion_id, s.starts_at
FROM migration.seance_map sm
JOIN public.seance s ON s.id = sm.internal_id
WHERE sm.source = 'webdfd'
  AND sm.last_seen_at < @cycle_start
  AND s.cancelled_at IS NULL;

-- name: MarkSeancesAnnulees :execrows
-- L'annulation est un MARQUAGE, jamais un DELETE : pointage,
-- justification_seance et presence_ledger référencent seance, et les deux
-- dernières en ON DELETE RESTRICT. Une feuille de présence déjà émise doit
-- rester lisible même si le cours a disparu du planning amont.
UPDATE public.seance
SET cancelled_at = now()
WHERE id = ANY(@ids::bigint[]) AND cancelled_at IS NULL;

-- name: UpsertSeanceMap :exec
INSERT INTO migration.seance_map (internal_id, source, external_id, last_seen_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (source, external_id) DO UPDATE
  SET internal_id  = EXCLUDED.internal_id,
      last_seen_at = now();

-- name: UpsertEleveGroupe :exec
INSERT INTO public.eleve_groupe (num_etudiant, id_groupe) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: GetElevesGroupe :many
SELECT num_etudiant FROM public.eleve_groupe WHERE id_groupe = $1;

-- name: DeleteEleveGroupeAbsents :exec
DELETE FROM public.eleve_groupe WHERE id_groupe = $1 AND num_etudiant != ALL($2::integer[]);
