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

-- name: UpdateSession :execrows
UPDATE session
SET name = sqlc.arg(name),
    given_name = sqlc.arg(given_name),
    family_name = sqlc.arg(family_name),
    preferred_user_name = sqlc.arg(preferred_user_name),
    email = sqlc.arg(email),
    groups = sqlc.arg(groups),
    access_token = sqlc.arg(access_token),
    refresh_token = sqlc.arg(refresh_token),
    token_expiry = sqlc.arg(token_expiry)
WHERE id = sqlc.arg(id)
AND subject = sqlc.arg(subject);
