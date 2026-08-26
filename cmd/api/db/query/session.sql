-- name: CreateSession :one
INSERT INTO session (subject, name, given_name, family_name, preferred_user_name, email, groups, access_token, refresh_token, token_expiry)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: GetSession :one
SELECT * FROM session
WHERE id = sqlc.arg(id)
AND created_at > sqlc.arg(session_expiration);

-- name: DeleteSession :exec
DELETE FROM session
WHERE id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM session
WHERE created_at <= sqlc.arg(expired_before);

-- name: TryAcquireCleanupLock :one
-- The first key identifies Konfidence; the second identifies auth cleanup.
SELECT pg_try_advisory_xact_lock(1263423054, 1);