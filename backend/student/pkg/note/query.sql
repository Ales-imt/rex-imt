-- name: GetNotesByEleve :many

SELECT 
    p.nom                  AS promo,
    ex.nom                 AS matiere,
    te.nom                 AS type_exercice,
    ex.date                AS date_exercice,
    r.noteobtenue,
    r.base,
    ex.coefficient,
    r.Commentaire

FROM eleves e

JOIN Realisation r      ON r.EVCLEUNIK = e.EVCLEUNIK
JOIN Exercice ex        ON ex.CTCLEUNIK = r.CTCLEUNIK
JOIN Promos_eleves pe   ON pe.EVCLEUNIK = e.EVCLEUNIK
JOIN promos p           ON p.P0CLEUNIK  = pe.P0CLEUNIK
                       AND p.P0CLEUNIK  = ex.P0CLEUNIK
LEFT JOIN TypeExercice te ON te.IDTypeExercice = ex.IDTypeExercice

WHERE e.mel = sqlc.arg(student_mail) 
  AND r.noteobtenue >= 0   -- exclut les notes non saisies (-1 par défaut)

ORDER BY p.nom, ex.date, ex.nom;