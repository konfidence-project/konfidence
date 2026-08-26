-- +goose Up
CREATE TABLE session (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    subject TEXT NOT NULL,
    name TEXT,
    given_name TEXT,
    family_name TEXT,
    preferred_user_name TEXT,
    email TEXT,
    groups TEXT[],
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_expiry BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE oidc_state (
    state TEXT PRIMARY KEY,
    nonce TEXT NOT NULL,
    return_url TEXT NOT NULL,
    code_verifier TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    client_code_challenge TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE oidc_exchange (
   code TEXT PRIMARY KEY,
   session_id UUID NOT NULL REFERENCES session(id) ON DELETE CASCADE,
   code_challenge TEXT NOT NULL,
   expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX session_created_at_idx ON session (created_at);
CREATE INDEX oidc_state_expires_at_idx ON oidc_state (expires_at);
CREATE INDEX oidc_exchange_expires_at_idx ON oidc_exchange (expires_at);

-- +goose Down
DROP TABLE IF EXISTS oidc_exchange;
DROP TABLE IF EXISTS oidc_state;
DROP TABLE IF EXISTS session;