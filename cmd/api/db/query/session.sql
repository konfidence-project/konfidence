-- name: CreateSession :one
INSERT INTO session (subject, name, given_name, family_name, preferred_user_name, email, groups, access_token, refresh_token, expiry)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: GetSession :one
SELECT * FROM session
WHERE id = $1 LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM session
WHERE id = $1;
