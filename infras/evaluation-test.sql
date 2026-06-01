-- =============================================================================
-- Données de test : évaluations de cours
-- Requiert : migrations v0.0.1 appliquées (tables eval_* et seed eval_chip_ref)
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- 0. Nettoyage (dans l'ordre inverse des FK)
-- ---------------------------------------------------------------------------
TRUNCATE eval_synthese_ia   CASCADE;
TRUNCATE eval_verbatim      CASCADE;
TRUNCATE eval_chip_reponse  CASCADE;
TRUNCATE eval_scores        CASCADE;
TRUNCATE eval_session       CASCADE;
TRUNCATE matiere            CASCADE;
TRUNCATE periode            CASCADE;
TRUNCATE promotion          CASCADE;
TRUNCATE annee_academique   CASCADE;

-- ---------------------------------------------------------------------------
-- 1. Années académiques  (bigserial → IDs 1 et 2)
-- ---------------------------------------------------------------------------
INSERT INTO annee_academique (libelle, debut, fin, active) VALUES
    ('2023-2024', '2023-09-01 00:00:00+02', '2024-08-31 23:59:59+02', false),
    ('2024-2025', '2024-09-01 00:00:00+02', '2025-08-31 23:59:59+02', true);

-- ---------------------------------------------------------------------------
-- 2. Promotions  (bigint explicite)
-- ---------------------------------------------------------------------------
INSERT INTO promotion (id, name, annee_academique_id) VALUES
    (1, 'M1 Informatique', 2),   -- année 2024-2025
    (2, 'M2 Informatique', 2),   -- année 2024-2025
    (3, 'M1 Informatique', 1);   -- année 2023-2024

-- ---------------------------------------------------------------------------
-- 3. Périodes  (bigserial → IDs 1 à 4)
-- ---------------------------------------------------------------------------
INSERT INTO periode (name, promotion_id) VALUES
    ('Semestre 1', 1),   -- id = 1  (M1 2024-2025)
    ('Semestre 2', 1),   -- id = 2
    ('Semestre 1', 2),   -- id = 3  (M2 2024-2025)
    ('Semestre 1', 3);   -- id = 4  (M1 2023-2024)

-- ---------------------------------------------------------------------------
-- 4. Matières  (bigint explicite)
-- Statuts attendus après seed des sessions :
--   1 - Algorithmique avancée  → 7 répondants → OK
--   2 - Base de données        → 3 répondants → WARN
--   3 - Réseaux & Sécurité     → 0 répondant  → NONE
--   4 - Machine Learning       → 8 répondants → OK
--   5 - Anglais professionnel  → 2 répondants → WARN
-- ---------------------------------------------------------------------------
INSERT INTO matiere (id, name, periode_id) VALUES
    (1, 'Algorithmique avancée',  1),
    (2, 'Base de données',        1),
    (3, 'Réseaux & Sécurité',     2),
    (4, 'Machine Learning',       3),
    (5, 'Anglais professionnel',  3);

-- ---------------------------------------------------------------------------
-- 5. Sessions d'évaluation
-- ---------------------------------------------------------------------------

-- Algorithmique avancée — 7 répondants (OK)
INSERT INTO eval_session (id, anon_id, matiere_id, format_suivi, tendance_semestre, assiduite, submitted_at) VALUES
    ('a1000001-0000-0000-0000-000000000001', 'anon-a1', 1, 'PRESENTIEL', 'PROGRES', 'TOUTES',             now() - interval '2 days'),
    ('a1000001-0000-0000-0000-000000000002', 'anon-a2', 1, 'PRESENTIEL', 'STABLE',  'TOUTES',             now() - interval '3 days'),
    ('a1000001-0000-0000-0000-000000000003', 'anon-a3', 1, 'PRESENTIEL', 'PROGRES', 'QUELQUES_ABSENCES',  now() - interval '3 days'),
    ('a1000001-0000-0000-0000-000000000004', 'anon-a4', 1, 'DISTANCIEL', 'STABLE',  'TOUTES',             now() - interval '4 days'),
    ('a1000001-0000-0000-0000-000000000005', 'anon-a5', 1, 'PRESENTIEL', 'PROGRES', 'TOUTES',             now() - interval '4 days'),
    ('a1000001-0000-0000-0000-000000000006', 'anon-a6', 1, 'HYBRIDE',    'BAISSE',  'BEAUCOUP',           now() - interval '5 days'),
    ('a1000001-0000-0000-0000-000000000007', 'anon-a7', 1, 'PRESENTIEL', 'STABLE',  'TOUTES',             now() - interval '5 days');

-- Base de données — 3 répondants (WARN)
INSERT INTO eval_session (id, anon_id, matiere_id, format_suivi, tendance_semestre, assiduite, submitted_at) VALUES
    ('b2000002-0000-0000-0000-000000000001', 'anon-b1', 2, 'PRESENTIEL', 'BAISSE',  'QUELQUES_ABSENCES',  now() - interval '1 day'),
    ('b2000002-0000-0000-0000-000000000002', 'anon-b2', 2, 'PRESENTIEL', 'STABLE',  'TOUTES',             now() - interval '2 days'),
    ('b2000002-0000-0000-0000-000000000003', 'anon-b3', 2, 'DISTANCIEL', 'BAISSE',  'BEAUCOUP',           now() - interval '3 days');

-- Réseaux & Sécurité — 0 répondant (NONE) : aucune session

-- Machine Learning — 8 répondants (OK)
INSERT INTO eval_session (id, anon_id, matiere_id, format_suivi, tendance_semestre, assiduite, submitted_at) VALUES
    ('d4000004-0000-0000-0000-000000000001', 'anon-d1', 4, 'PRESENTIEL', 'PROGRES', 'TOUTES',             now() - interval '1 day'),
    ('d4000004-0000-0000-0000-000000000002', 'anon-d2', 4, 'PRESENTIEL', 'PROGRES', 'TOUTES',             now() - interval '1 day'),
    ('d4000004-0000-0000-0000-000000000003', 'anon-d3', 4, 'PRESENTIEL', 'STABLE',  'TOUTES',             now() - interval '2 days'),
    ('d4000004-0000-0000-0000-000000000004', 'anon-d4', 4, 'DISTANCIEL', 'PROGRES', 'QUELQUES_ABSENCES',  now() - interval '2 days'),
    ('d4000004-0000-0000-0000-000000000005', 'anon-d5', 4, 'PRESENTIEL', 'PROGRES', 'TOUTES',             now() - interval '3 days'),
    ('d4000004-0000-0000-0000-000000000006', 'anon-d6', 4, 'HYBRIDE',    'STABLE',  'TOUTES',             now() - interval '3 days'),
    ('d4000004-0000-0000-0000-000000000007', 'anon-d7', 4, 'PRESENTIEL', 'PROGRES', 'TOUTES',             now() - interval '4 days'),
    ('d4000004-0000-0000-0000-000000000008', 'anon-d8', 4, 'PRESENTIEL', 'STABLE',  'TOUTES',             now() - interval '4 days');

-- Anglais professionnel — 2 répondants (WARN)
INSERT INTO eval_session (id, anon_id, matiere_id, format_suivi, tendance_semestre, assiduite, submitted_at) VALUES
    ('e5000005-0000-0000-0000-000000000001', 'anon-e1', 5, 'PRESENTIEL', 'STABLE',  'TOUTES',             now() - interval '2 days'),
    ('e5000005-0000-0000-0000-000000000002', 'anon-e2', 5, 'DISTANCIEL', 'BAISSE',  'QUELQUES_ABSENCES',  now() - interval '3 days');

-- ---------------------------------------------------------------------------
-- 6. Scores
-- ---------------------------------------------------------------------------

-- Algorithmique avancée — bons scores, NPS élevés
INSERT INTO eval_scores (session_id, score_peda_global, score_peda_clarte, score_contenu_adequation, charge_perception, charge_heures_semaine, score_difficulte, score_supports, score_ambiance, nps) VALUES
    ('a1000001-0000-0000-0000-000000000001', 5, 4, 5, 'ADAPTEE', 'H_3_5',    4, 4, 5, 9),
    ('a1000001-0000-0000-0000-000000000002', 4, 4, 4, 'ADAPTEE', 'H_1_3',    3, 4, 4, 8),
    ('a1000001-0000-0000-0000-000000000003', 5, 5, 5, 'LEGERE',  'H_1_3',    4, 5, 5, 10),
    ('a1000001-0000-0000-0000-000000000004', 4, 3, 4, 'ADAPTEE', 'H_3_5',    4, 3, 4, 7),
    ('a1000001-0000-0000-0000-000000000005', 5, 4, 5, 'ADAPTEE', 'H_1_3',    3, 4, 5, 9),
    ('a1000001-0000-0000-0000-000000000006', 3, 3, 3, 'LOURDE',  'H_PLUS_5', 5, 3, 3, 5),
    ('a1000001-0000-0000-0000-000000000007', 4, 4, 4, 'ADAPTEE', 'H_3_5',    4, 4, 4, 8);

-- Base de données — scores faibles, charge lourde
INSERT INTO eval_scores (session_id, score_peda_global, score_peda_clarte, score_contenu_adequation, charge_perception, charge_heures_semaine, score_difficulte, score_supports, score_ambiance, nps) VALUES
    ('b2000002-0000-0000-0000-000000000001', 2, 2, 3, 'LOURDE',  'H_PLUS_5', 5, 2, 3, 4),
    ('b2000002-0000-0000-0000-000000000002', 3, 3, 3, 'ADAPTEE', 'H_3_5',   4, 3, 3, 6),
    ('b2000002-0000-0000-0000-000000000003', 2, 2, 2, 'LOURDE',  'H_PLUS_5', 5, 2, 2, 3);

-- Machine Learning — très bons scores
INSERT INTO eval_scores (session_id, score_peda_global, score_peda_clarte, score_contenu_adequation, charge_perception, charge_heures_semaine, score_difficulte, score_supports, score_ambiance, nps) VALUES
    ('d4000004-0000-0000-0000-000000000001', 5, 5, 5, 'ADAPTEE', 'H_3_5',    4, 5, 5, 10),
    ('d4000004-0000-0000-0000-000000000002', 5, 4, 5, 'ADAPTEE', 'H_3_5',    4, 4, 5, 9),
    ('d4000004-0000-0000-0000-000000000003', 4, 4, 4, 'ADAPTEE', 'H_1_3',    3, 4, 4, 8),
    ('d4000004-0000-0000-0000-000000000004', 5, 5, 5, 'LEGERE',  'H_1_3',    3, 5, 5, 10),
    ('d4000004-0000-0000-0000-000000000005', 4, 4, 4, 'ADAPTEE', 'H_3_5',    4, 4, 4, 8),
    ('d4000004-0000-0000-0000-000000000006', 5, 5, 5, 'ADAPTEE', 'H_3_5',    3, 5, 5, 9),
    ('d4000004-0000-0000-0000-000000000007', 4, 4, 4, 'LOURDE',  'H_PLUS_5', 5, 4, 4, 7),
    ('d4000004-0000-0000-0000-000000000008', 5, 5, 5, 'ADAPTEE', 'H_1_3',    3, 5, 5, 10);

-- Anglais professionnel — scores passables
INSERT INTO eval_scores (session_id, score_peda_global, score_peda_clarte, score_contenu_adequation, charge_perception, charge_heures_semaine, score_difficulte, score_supports, score_ambiance, nps) VALUES
    ('e5000005-0000-0000-0000-000000000001', 3, 3, 3, 'ADAPTEE', 'H_MOINS_1', 2, 3, 3, 6),
    ('e5000005-0000-0000-0000-000000000002', 2, 2, 3, 'ADAPTEE', 'H_MOINS_1', 2, 2, 3, 5);

-- ---------------------------------------------------------------------------
-- 7. Chips cochées  (IDs proviennent du seed 014-create-evaluation.yaml)
-- ---------------------------------------------------------------------------

-- Algorithmique — points positifs dominants, 1 signalement de rythme
INSERT INTO eval_chip_reponse (session_id, chip_id) VALUES
    ('a1000001-0000-0000-0000-000000000001', 'EXEMPLES_CONCRETS'),
    ('a1000001-0000-0000-0000-000000000001', 'CAS_PRATIQUES'),
    ('a1000001-0000-0000-0000-000000000001', 'DISPONIBLE'),
    ('a1000001-0000-0000-0000-000000000002', 'EXEMPLES_CONCRETS'),
    ('a1000001-0000-0000-0000-000000000002', 'THEORIE_SOLIDE'),
    ('a1000001-0000-0000-0000-000000000002', 'DYNAMIQUE'),
    ('a1000001-0000-0000-0000-000000000003', 'EXEMPLES_CONCRETS'),
    ('a1000001-0000-0000-0000-000000000003', 'CAS_PRATIQUES'),
    ('a1000001-0000-0000-0000-000000000003', 'BIENVEILLANT'),
    ('a1000001-0000-0000-0000-000000000004', 'THEORIE_SOLIDE'),
    ('a1000001-0000-0000-0000-000000000004', 'A_ECOUTE'),
    ('a1000001-0000-0000-0000-000000000005', 'CAS_PRATIQUES'),
    ('a1000001-0000-0000-0000-000000000005', 'EXEMPLES_CONCRETS'),
    ('a1000001-0000-0000-0000-000000000006', 'TROP_DENSE'),
    ('a1000001-0000-0000-0000-000000000006', 'RYTHME_IMPOSE'),
    ('a1000001-0000-0000-0000-000000000007', 'EXEMPLES_CONCRETS'),
    ('a1000001-0000-0000-0000-000000000007', 'DYNAMIQUE');

-- Base de données — points négatifs dominants
INSERT INTO eval_chip_reponse (session_id, chip_id) VALUES
    ('b2000002-0000-0000-0000-000000000001', 'TROP_THEORIQUE'),
    ('b2000002-0000-0000-0000-000000000001', 'SLIDES_ILLISIBLES'),
    ('b2000002-0000-0000-0000-000000000001', 'PEU_DISPONIBLE'),
    ('b2000002-0000-0000-0000-000000000002', 'TROP_DENSE'),
    ('b2000002-0000-0000-0000-000000000002', 'INCOMPLET'),
    ('b2000002-0000-0000-0000-000000000003', 'TROP_THEORIQUE'),
    ('b2000002-0000-0000-0000-000000000003', 'PAS_A_JOUR'),
    ('b2000002-0000-0000-0000-000000000003', 'PEU_ECOUTE');

-- Machine Learning — points positifs très forts
INSERT INTO eval_chip_reponse (session_id, chip_id) VALUES
    ('d4000004-0000-0000-0000-000000000001', 'CAS_PRATIQUES'),
    ('d4000004-0000-0000-0000-000000000001', 'ACTUALITE'),
    ('d4000004-0000-0000-0000-000000000001', 'DISPONIBLE'),
    ('d4000004-0000-0000-0000-000000000002', 'CAS_PRATIQUES'),
    ('d4000004-0000-0000-0000-000000000002', 'EXEMPLES_CONCRETS'),
    ('d4000004-0000-0000-0000-000000000002', 'BIENVEILLANT'),
    ('d4000004-0000-0000-0000-000000000003', 'ACTUALITE'),
    ('d4000004-0000-0000-0000-000000000003', 'THEORIE_SOLIDE'),
    ('d4000004-0000-0000-0000-000000000003', 'A_ECOUTE'),
    ('d4000004-0000-0000-0000-000000000004', 'CAS_PRATIQUES'),
    ('d4000004-0000-0000-0000-000000000004', 'EXEMPLES_CONCRETS'),
    ('d4000004-0000-0000-0000-000000000004', 'DYNAMIQUE'),
    ('d4000004-0000-0000-0000-000000000005', 'CAS_PRATIQUES'),
    ('d4000004-0000-0000-0000-000000000005', 'ACTUALITE'),
    ('d4000004-0000-0000-0000-000000000006', 'EXEMPLES_CONCRETS'),
    ('d4000004-0000-0000-0000-000000000006', 'THEORIE_SOLIDE'),
    ('d4000004-0000-0000-0000-000000000007', 'TROP_DENSE'),
    ('d4000004-0000-0000-0000-000000000008', 'CAS_PRATIQUES'),
    ('d4000004-0000-0000-0000-000000000008', 'BIENVEILLANT');

-- ---------------------------------------------------------------------------
-- 8. Verbatims
-- ---------------------------------------------------------------------------

-- Algorithmique avancée
INSERT INTO eval_verbatim (id, session_id, dimension, texte, created_at) VALUES
    (gen_random_uuid(), 'a1000001-0000-0000-0000-000000000001', 'NPS',      'Excellente matière, très bien structurée. Les exemples pratiques étaient vraiment utiles pour ancrer les concepts.',                              now() - interval '2 days'),
    (gen_random_uuid(), 'a1000001-0000-0000-0000-000000000003', 'SUPPORTS', 'Les slides sont clairs et bien organisés, très faciles à utiliser pour réviser.',                                                                 now() - interval '3 days'),
    (gen_random_uuid(), 'a1000001-0000-0000-0000-000000000006', 'AMBIANCE', 'Le rythme est parfois trop soutenu. Difficile de suivre quand on a quelques absences.',                                                           now() - interval '5 days'),
    (gen_random_uuid(), 'a1000001-0000-0000-0000-000000000006', 'NPS',      'Trop de contenu brassé en trop peu de temps. Il faudrait soit alléger le programme soit ajouter des séances de TD.',                             now() - interval '5 days');

-- Base de données
INSERT INTO eval_verbatim (id, session_id, dimension, texte, created_at) VALUES
    (gen_random_uuid(), 'b2000002-0000-0000-0000-000000000001', 'SUPPORTS', 'Les slides sont illisibles : police trop petite et trop de texte par page. Impossible à relire pour préparer l''examen.',                        now() - interval '1 day'),
    (gen_random_uuid(), 'b2000002-0000-0000-0000-000000000001', 'AMBIANCE', 'Le professeur n''est pas disponible pour répondre aux questions en dehors des séances. Les mails restent sans réponse pendant des semaines.',     now() - interval '1 day'),
    (gen_random_uuid(), 'b2000002-0000-0000-0000-000000000002', 'NPS',      'Le cours est beaucoup trop théorique sans lien concret avec la pratique réelle. Il manque des TP ou mini-projets.',                               now() - interval '2 days'),
    (gen_random_uuid(), 'b2000002-0000-0000-0000-000000000003', 'SUPPORTS', 'Le contenu n''est pas à jour. Certains outils et syntaxes présentés sont obsolètes ou ont été remplacés depuis plusieurs années.',                now() - interval '3 days');

-- Machine Learning
INSERT INTO eval_verbatim (id, session_id, dimension, texte, created_at) VALUES
    (gen_random_uuid(), 'd4000004-0000-0000-0000-000000000001', 'NPS',      'Meilleure matière du semestre sans hésitation. Les projets pratiques sont vraiment motivants et appliqués à des cas réels.',                     now() - interval '1 day'),
    (gen_random_uuid(), 'd4000004-0000-0000-0000-000000000002', 'SUPPORTS', 'Les notebooks Jupyter fournis sont excellents, très bien documentés et progressifs. On peut les réutiliser après le cours.',                      now() - interval '1 day'),
    (gen_random_uuid(), 'd4000004-0000-0000-0000-000000000004', 'AMBIANCE', 'Prof très disponible, répond toujours rapidement aux questions sur le forum et en dehors des cours. Très bienveillant.',                          now() - interval '2 days'),
    (gen_random_uuid(), 'd4000004-0000-0000-0000-000000000007', 'NPS',      'Matière intense mais très enrichissante. La charge de travail est élevée mais justifiée par la densité des sujets abordés.',                      now() - interval '4 days');

-- ---------------------------------------------------------------------------
-- 9. Synthèse IA  (Machine Learning — statut VALIDEE)
-- ---------------------------------------------------------------------------
INSERT INTO eval_synthese_ia (
    id, matiere_id, semestre, nb_repondants,
    donnees_snapshot, synthese_texte, statut, generated_at
) VALUES (
    gen_random_uuid(),
    4,
    'Semestre 1 2024-2025',
    8,
    '{"score_peda": 4.63, "score_clarte": 4.50, "score_contenu": 4.75, "score_supports": 4.63, "score_ambiance": 4.75, "nps_moyen": 8.88}',
    'La matière Machine Learning obtient d''excellents retours de la part des 8 étudiants ayant soumis leur évaluation.

**Points forts**
Le score pédagogique moyen de 4,6/5 reflète une grande satisfaction vis-à-vis de la qualité de l''enseignement. Les cas pratiques sont plébiscités par 75 % des répondants, suivis de l''actualité des contenus (50 %) et de la disponibilité du professeur (38 %). Les supports de cours (notebooks Jupyter) sont décrits comme "excellents et bien documentés" par plusieurs étudiants.

**NPS**
Le NPS moyen de 8,9 est remarquable : 75 % de promoteurs, 12,5 % de passifs, 12,5 % de détracteurs. Ce score place cette matière parmi les meilleures du programme.

**Points de vigilance**
Un étudiant sur huit signale une charge de travail lourde (LOURDE) avec plus de 5 heures par semaine. Ce signal reste marginal mais mérite attention si le programme s''intensifie.

**Recommandation**
Maintenir le format actuel en insistant sur les projets pratiques. Envisager une séance dédiée à la gestion du temps pour les étudiants moins à l''aise avec la charge.',
    'VALIDEE',
    now() - interval '12 hours'
);

COMMIT;

-- ---------------------------------------------------------------------------
-- Vérification rapide : dot_status attendus
-- ---------------------------------------------------------------------------
SELECT
    m.id,
    m.name                                                                  AS matiere,
    pe.name                                                                 AS periode,
    pr.name                                                                 AS promotion,
    COUNT(es.id) FILTER (WHERE es.submitted_at IS NOT NULL)                AS nb_repondants,
    CASE
        WHEN COUNT(es.id) FILTER (WHERE es.submitted_at IS NOT NULL) >= 5 THEN 'OK'
        WHEN COUNT(es.id) FILTER (WHERE es.submitted_at IS NOT NULL) >= 1 THEN 'WARN'
        ELSE 'NONE'
    END                                                                     AS dot_status
FROM matiere m
JOIN periode pe ON pe.id = m.periode_id
JOIN promotion pr ON pr.id = pe.promotion_id
LEFT JOIN eval_session es ON es.matiere_id = m.id
GROUP BY m.id, m.name, pe.name, pr.name
ORDER BY pr.name, pe.name, m.name;
