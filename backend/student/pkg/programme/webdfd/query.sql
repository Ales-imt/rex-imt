-- name: GetWebdfdStudentExternalID :one
-- Résolution email → identifiant webdfd d'un ÉTUDIANT (TYPECLE=evcleunik).
SELECT um.external_id
FROM migration.user_map um
JOIN public."user" u ON u.id = um.internal_id
WHERE u.email = $1 AND um.source = 'webdfd';

-- name: GetWebdfdProfExternalID :one
-- Résolution email → identifiant webdfd d'un PROFESSEUR (TYPECLE=prcleunik),
-- consultée en repli lorsque l'email n'est pas dans user_map.
SELECT pm.external_id
FROM migration.prof_map pm
JOIN public."user" u ON u.id = pm.internal_id
WHERE u.email = $1 AND pm.source = 'webdfd';
