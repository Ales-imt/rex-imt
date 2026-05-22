-- name: GetElevesSortis :many
SELECT e.mel, MAX(p.datefin) AS derniere_fin
FROM eleves e
JOIN Promos_eleves pe ON pe.EVCLEUNIK = e.EVCLEUNIK
JOIN promos p         ON p.P0CLEUNIK  = pe.P0CLEUNIK
WHERE e.mel IS NOT NULL AND e.mel != ''
GROUP BY e.mel
HAVING MAX(p.datefin) < NOW()
   AND MAX(p.datefin) < NOW() - INTERVAL 1 YEAR;
