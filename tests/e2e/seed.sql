-- Seed E2E : environnement de test déterministe pour la stack locale.
--
-- Complète la base issue de `make local` (liquibase appliqué, tables vides ou
-- repeuplées par le sync) avec un jeu MINIMAL et STABLE :
--   - des comptes alignés sur l'annuaire LDAP de test
--     (infras/container/ldap/bootstrap/02-users.ldif, mot de passe « password ») ;
--   - une promotion/période/matière/groupe et des séances de la semaine
--     COURANTE, dont une avec groupe et remarque — ce que les tests Playwright
--     vérifient à l'écran.
--
-- Rejouable : chaque bloc upserte par clé naturelle (email, name), jamais par
-- id. Les séances de test sont reconnaissables à leur salle 'E2E-*' et
-- re-créées à chaque exécution sur la semaine en cours.
--
-- Le sync ne touche pas à ces lignes : l'annulation des séances disparues ne
-- considère que celles présentes dans migration.seance_map, où celles-ci ne
-- figurent pas.

BEGIN;

-- ── Comptes ──────────────────────────────────────────────────────────────────
-- PostLdap exige l'utilisateur en base AVANT le login : le LDAP n'authentifie
-- que le mot de passe, les rôles viennent d'ici.

INSERT INTO public."user" (name, surname, email, roles) VALUES
  ('TRENS',   'Clement', 'clement.trens@etu.mines-ales.fr', '{ELEVE}'),
  ('DUPONT',  'Bob',     'bob.dupont@etu.mines-ales.fr',    '{ELEVE}'),
  ('LECOCQ',  'Claire',  'claire.lecocq@mines-ales.fr',     '{ADMIN,GESTIONNAIRE,ELEVE}'),
  ('BERNARD', 'Jean',    'jean.bernard@mines-ales.fr',      '{PROF}')
ON CONFLICT (email) DO UPDATE SET roles = EXCLUDED.roles;

-- ── Référentiel planning ─────────────────────────────────────────────────────

INSERT INTO public.promotion (name, annee_id)
SELECT '1A TEST E2E', a.id FROM public.annee a WHERE a.name = '2026/2027'
ON CONFLICT DO NOTHING;

INSERT INTO public.periode (name, promotion_id, annee)
SELECT 'S5', p.id, 2026 FROM public.promotion p
WHERE p.name = '1A TEST E2E'
  AND NOT EXISTS (
    SELECT 1 FROM public.periode pe WHERE pe.promotion_id = p.id AND pe.name = 'S5'
  );

INSERT INTO public.matiere (name, periode_id, annee)
SELECT '5.0 / MATIERE TEST E2E (1A TEST)', pe.id, 2026
FROM public.periode pe JOIN public.promotion p ON p.id = pe.promotion_id
WHERE p.name = '1A TEST E2E' AND pe.name = 'S5'
  AND NOT EXISTS (SELECT 1 FROM public.matiere m WHERE m.name = '5.0 / MATIERE TEST E2E (1A TEST)');

INSERT INTO public.groupe (name, taille, promo_id)
SELECT 'G1-E2E', 2, p.id FROM public.promotion p
WHERE p.name = '1A TEST E2E'
  AND NOT EXISTS (
    SELECT 1 FROM public.groupe g WHERE g.promo_id = p.id AND g.name = 'G1-E2E'
  );

-- Les deux élèves de test appartiennent au groupe.
INSERT INTO public.eleve_groupe (num_etudiant, id_groupe)
SELECT u.id, g.id
FROM public."user" u,
     public.groupe g JOIN public.promotion p ON p.id = g.promo_id
WHERE p.name = '1A TEST E2E' AND g.name = 'G1-E2E'
  AND u.email IN ('clement.trens@etu.mines-ales.fr', 'bob.dupont@etu.mines-ales.fr')
ON CONFLICT DO NOTHING;

-- ── Séances du jour ──────────────────────────────────────────────────────────
-- Recréées à chaque exécution, ancrées sur le JOUR courant : l'écran élève
-- s'ouvre sur aujourd'hui, les tests n'ont donc jamais à naviguer. Les horaires
-- sont exprimés à Paris, comme ceux écrits par la synchronisation.

DELETE FROM public.seance WHERE salle LIKE 'E2E-%';

-- Séance de groupe, avec remarque : le cas complet, celui que les tests
-- regardent en priorité (salle · prof · groupe + remarque en italique).
INSERT INTO public.seance (matiere_id, starts_at, ends_at, salle, prof, promotion_id, groupe_id, prof_id, remarque, opened_at)
SELECT m.id,
       date_trunc('day', (now() AT TIME ZONE 'Europe/Paris'))::timestamp AT TIME ZONE 'Europe/Paris' + interval '9 hours',
       date_trunc('day', (now() AT TIME ZONE 'Europe/Paris'))::timestamp AT TIME ZONE 'Europe/Paris' + interval '11 hours',
       'E2E-B101', 'M. BERNARD', p.id, g.id, u.id,
       'Réunion DRDV — seed E2E',
       now()
FROM public.matiere m
JOIN public.periode pe ON pe.id = m.periode_id
JOIN public.promotion p ON p.id = pe.promotion_id
JOIN public.groupe g ON g.promo_id = p.id AND g.name = 'G1-E2E'
JOIN public."user" u ON u.email = 'jean.bernard@mines-ales.fr'
WHERE p.name = '1A TEST E2E' AND m.name LIKE '%TEST E2E%';

-- Séance de promotion entière (groupe_id NULL), sans remarque : le cas nominal,
-- qui vérifie a contrario que rien ne s'affiche en trop.
INSERT INTO public.seance (matiere_id, starts_at, ends_at, salle, prof, promotion_id, groupe_id, prof_id, remarque, opened_at)
SELECT m.id,
       date_trunc('day', (now() AT TIME ZONE 'Europe/Paris'))::timestamp AT TIME ZONE 'Europe/Paris' + interval '14 hours',
       date_trunc('day', (now() AT TIME ZONE 'Europe/Paris'))::timestamp AT TIME ZONE 'Europe/Paris' + interval '16 hours',
       'E2E-A202', 'M. BERNARD', p.id, NULL, u.id,
       NULL,
       now()
FROM public.matiere m
JOIN public.periode pe ON pe.id = m.periode_id
JOIN public.promotion p ON p.id = pe.promotion_id
JOIN public."user" u ON u.email = 'jean.bernard@mines-ales.fr'
WHERE p.name = '1A TEST E2E' AND m.name LIKE '%TEST E2E%';

-- Séance du lendemain : alimente le point (dot) du jour suivant dans la barre
-- de semaine côté élève.
INSERT INTO public.seance (matiere_id, starts_at, ends_at, salle, prof, promotion_id, groupe_id, prof_id, remarque, opened_at)
SELECT m.id,
       date_trunc('day', (now() AT TIME ZONE 'Europe/Paris'))::timestamp AT TIME ZONE 'Europe/Paris' + interval '1 day 8 hours',
       date_trunc('day', (now() AT TIME ZONE 'Europe/Paris'))::timestamp AT TIME ZONE 'Europe/Paris' + interval '1 day 10 hours',
       'E2E-C303', 'M. BERNARD', p.id, g.id, u.id,
       'Cours/TD 3 — seed E2E',
       now()
FROM public.matiere m
JOIN public.periode pe ON pe.id = m.periode_id
JOIN public.promotion p ON p.id = pe.promotion_id
JOIN public.groupe g ON g.promo_id = p.id AND g.name = 'G1-E2E'
JOIN public."user" u ON u.email = 'jean.bernard@mines-ales.fr'
WHERE p.name = '1A TEST E2E' AND m.name LIKE '%TEST E2E%';

COMMIT;
