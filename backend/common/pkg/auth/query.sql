
-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token, expires_at, created_at, revoked, session, token_version)
        VALUES (@userId, @token, @expire, @created, @revoked, @session, @token_version);

-- name: GetRefreshToken :one   
SELECT *
        FROM refresh_tokens
        WHERE token = @token;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE token = @token;

-- name: GetRefreshTokenBySession :one
SELECT *
        FROM refresh_tokens
        WHERE session = @session AND revoked = false;

-- name: DeleteRefreshTokenBySession :exec
DELETE FROM refresh_tokens WHERE session = @session;

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








