-- name: GetAnneeByDate :one
SELECT id, debut, fin FROM public.annee WHERE debut <= $1 AND fin >= $1;

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
INSERT INTO public.student (user_id) VALUES ($1);

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

-- name: GetSeanceBySource :one
SELECT internal_id FROM migration.seance_map WHERE source = $1 AND external_id = $2;

-- name: CreateSeance :one
INSERT INTO public.seance
  (matiere_id, starts_at, ends_at, salle, prof, promotion_id, groupe_id, prof_id, opened_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $2)
RETURNING id;

-- name: UpdateSeance :exec
UPDATE public.seance
SET matiere_id   = $2,
    starts_at    = $3,
    ends_at      = $4,
    salle        = $5,
    prof         = $6,
    promotion_id = $7,
    groupe_id    = $8,
    prof_id      = $9
WHERE id = $1;

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
