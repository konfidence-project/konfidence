-- name: CreateSession :one
INSERT INTO session (subject, name, given_name, family_name, preferred_user_name, email, groups, access_token, refresh_token, expiry)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: GetAndTouchSession :one
WITH delete_expired AS (
DELETE FROM session AS expired
WHERE expired.id = sqlc.arg(id)
AND expired.last_accessed_at <= sqlc.arg(session_expiry)
RETURNING expired.id
)
UPDATE session AS active
SET last_accessed_at = sqlc.arg(accessed_at)
WHERE active.id = sqlc.arg(id)
AND NOT EXISTS (SELECT 1 FROM delete_expired)
RETURNING active.*;

-- name: DeleteSession :exec
DELETE FROM session
WHERE id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM session
WHERE last_accessed_at <= sqlc.arg(expired_before);
