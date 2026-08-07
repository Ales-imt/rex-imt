-- Sorties d'étudiants lues dans Auréga (source de vérité de la scolarité).
--
-- Le cycle de vie d'un compte se joue en DEUX temps, avec deux seuils distincts
-- ancrés sur la date de sortie (MAX(datefin)) :
--   1. départ + 1 an   → désactivation (accès coupé, identité conservée) ;
--   2. départ + 10 ans → anonymisation en place de l'identité.
--
-- L'ancrage sur datefin est une simplification : la durée de conservation des
-- pièces de présence court en toute rigueur à compter du dernier versement du
-- financeur, information dont l'école ne dispose pas ici. Voir docs/rgpd-dpo.md
-- §7 — point à revalider avec le DPO.
--
-- Le seuil est passé en paramètre (date calculée côté Go) plutôt qu'écrit en
-- INTERVAL littéral : une seule forme de requête, testable, et les deux
-- horizons restent définis au même endroit dans purge.go.

-- name: GetElevesSortisAvant :many
-- Étudiants dont la dernière promotion s'est terminée avant le seuil fourni.
-- L'email sert de clé de jointure avec la table "user" de PostgreSQL.
SELECT e.mel, MAX(p.datefin) AS derniere_fin
FROM eleves e
JOIN Promos_eleves pe ON pe.EVCLEUNIK = e.EVCLEUNIK
JOIN promos p         ON p.P0CLEUNIK  = pe.P0CLEUNIK
WHERE e.mel IS NOT NULL AND e.mel != ''
GROUP BY e.mel
HAVING MAX(p.datefin) < ?;

-- name: GetDateFinByEmail :one
-- Date de sortie d'un étudiant donné, pour décider au cas par cas (CRUD admin)
-- si l'horizon de conservation des pièces de présence est échu.
SELECT MAX(p.datefin) AS derniere_fin
FROM eleves e
JOIN Promos_eleves pe ON pe.EVCLEUNIK = e.EVCLEUNIK
JOIN promos p         ON p.P0CLEUNIK  = pe.P0CLEUNIK
WHERE e.mel = ?
GROUP BY e.mel;
