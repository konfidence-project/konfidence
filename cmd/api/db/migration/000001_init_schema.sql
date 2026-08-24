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

CREATE INDEX session_created_at_idx ON session (created_at);

-- +goose Down
DROP TABLE IF EXISTS session;