-- name: CreateSession :one
INSERT INTO session (
    subject,
    name,
    given_name,
    family_name,
    preferred_user_name,
    email,
    groups,
    access_token,
    refresh_token,
    token_expiry,
    expires_at
)
VALUES (
    sqlc.arg(subject),
    sqlc.arg(name),
    sqlc.arg(given_name),
    sqlc.arg(family_name),
    sqlc.arg(preferred_user_name),
    sqlc.arg(email),
    sqlc.arg(groups),
    sqlc.arg(access_token),
    sqlc.arg(refresh_token),
    sqlc.arg(token_expiry),
    NOW() + sqlc.arg(expiration)::interval
)
RETURNING id;

-- name: GetSession :one
SELECT * FROM session
WHERE id = sqlc.arg(id)
AND expires_at > NOW();


-- name: DeleteSession :exec
DELETE FROM session
WHERE id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM session
WHERE expires_at <= NOW();

-- name: TryAcquireCleanupLock :one
-- The first key identifies Konfidence; the second identifies auth cleanup.
SELECT pg_try_advisory_xact_lock(1263423054, 1);