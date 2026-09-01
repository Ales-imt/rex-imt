
-- name: CreateRefreshToken :exec
-- prev_token / prev_consumed_at : hachage du jeton que cette rotation vient de
-- consommer et instant de sa consommation — l'ancrage de la fenêtre de grâce
-- (cf. jetonEnGrace, jwt.go). NULL au login : pas de prédécesseur.
INSERT INTO refresh_tokens (user_id, token, expires_at, created_at, revoked, session, token_version, prev_token, prev_consumed_at)
        VALUES (@userId, @token, @expire, @created, @revoked, @session, @token_version, @prev_token, @prev_consumed_at);

-- name: GetRefreshToken :one   
SELECT *
        FROM refresh_tokens
        WHERE token = @token;

-- name: ConsumeRefreshToken :one
-- Consommation ATOMIQUE du jeton de renouvellement : suppression et lecture en
-- une seule instruction. De deux requêtes concurrentes portant le même jeton,
-- une seule obtient la ligne ; l'autre reçoit zéro ligne (ErrNoRows) et doit
-- refuser. Un SELECT puis DELETE séparés laissaient les deux passer.
DELETE FROM refresh_tokens WHERE token = @token RETURNING *;

-- name: GetRefreshTokenByPrev :one
-- Réémission idempotente : retrouve le jeton VIVANT dont le prédécesseur
-- direct est le jeton présenté. Lecture pure — la réémission n'écrit rien,
-- deux rejeux concurrents obtiennent la même réponse, c'est le but.
SELECT *
        FROM refresh_tokens
        WHERE prev_token = @prev_token;

-- name: GetRefreshTokenBySession :one
SELECT *
        FROM refresh_tokens
        WHERE session = @session AND revoked = false;

-- name: DeleteRefreshTokenBySession :exec
DELETE FROM refresh_tokens WHERE session = @session;

-- name: DeleteRefreshTokensByUser :exec
DELETE FROM refresh_tokens WHERE user_id = @user_id;

-- name: CleanUpTokens :exec
DELETE FROM refresh_tokens WHERE expires_at < NOW();

--  Ici pour eviter un import circulaire avec le package utilisateur.
-- name: GetUserById :one
SELECT id, version, name, surname, email, roles, blame
FROM public.user
WHERE id = @id AND disabled_at IS NULL;

--  Ici pour eviter un import circulaire avec le package utilisateur.
-- name: GetUserByMail :one
SELECT id,  version, name, surname, email, roles, blame
FROM public.user
WHERE email = @email AND disabled_at IS NULL;

-- name: GetUserByMailAndSource :one
SELECT id, version, name, surname, email, roles, blame
FROM public.user
WHERE email = @email AND auth_source = @auth_source AND disabled_at IS NULL;

-- name: CreateLoginCode :exec
INSERT INTO login_code (user_id, code_hash, expires_at, created_at)
	VALUES (@user_id, @code_hash, @expires_at, @created_at);

-- name: GetActiveLoginCodeByUser :one
SELECT * FROM login_code
WHERE user_id = @user_id AND consumed_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementLoginCodeAttempts :exec
UPDATE login_code SET attempts = attempts + 1 WHERE id = @id;

-- name: ConsumeLoginCode :exec
UPDATE login_code SET consumed_at = NOW() WHERE id = @id;

-- name: InvalidateLoginCodes :exec
UPDATE login_code SET consumed_at = NOW() WHERE user_id = @user_id AND consumed_at IS NULL;

-- name: CleanUpLoginCodes :exec
DELETE FROM login_code WHERE consumed_at IS NOT NULL OR expires_at < NOW();








