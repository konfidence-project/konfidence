-- name: SaveOIDCState :exec
INSERT INTO oidc_state (
    state,
    nonce,
    return_url,
    code_verifier,
    code_challenge_method,
    code_challenge,
    client_code_challenge,
    created_at,
    expires_at
)
VALUES (
    sqlc.arg(state),
    sqlc.arg(nonce),
    sqlc.arg(return_url),
    sqlc.arg(code_verifier),
    sqlc.arg(code_challenge_method),
    sqlc.arg(code_challenge),
    sqlc.arg(client_code_challenge),
    sqlc.arg(created_at),
    NOW() + sqlc.arg(expiration)::interval
);

-- name: ConsumeOIDCState :one
WITH consumed AS (DELETE FROM oidc_state WHERE state = sqlc.arg(state) RETURNING *)
SELECT * FROM consumed
WHERE expires_at > NOW();

-- name: DeleteExpiredOIDCStates :execrows
DELETE FROM oidc_state
WHERE expires_at <= NOW();

-- name: SaveOIDCExchange :exec
INSERT INTO oidc_exchange (
    code,
    session_id,
    code_challenge,
    expires_at
)
VALUES (
    sqlc.arg(code),
    sqlc.arg(session_id),
    sqlc.arg(code_challenge),
    NOW() + sqlc.arg(expiration)::interval
);

-- name: ConsumeOIDCExchange :one
WITH consumed AS (DELETE FROM oidc_exchange WHERE code = sqlc.arg(code) RETURNING *)
SELECT * FROM consumed
WHERE expires_at > NOW();

-- name: DeleteExpiredOIDCExchanges :execrows
DELETE FROM oidc_exchange
WHERE expires_at <= NOW();