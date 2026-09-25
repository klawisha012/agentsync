CREATE TABLE accounts (
    id text PRIMARY KEY,
    email text NOT NULL,
    email_key text NOT NULL,
    name text NOT NULL,
    name_key text NOT NULL,
    password_hash bytea NOT NULL,
    verified boolean NOT NULL DEFAULT false,
    confirm_token text,
    confirm_expires timestamptz,
    reset_token text,
    reset_expires timestamptz,
    CONSTRAINT accounts_email_key_unique UNIQUE (email_key),
    CONSTRAINT accounts_name_key_unique UNIQUE (name_key)
);

CREATE TABLE sessions (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_account_id_idx ON sessions (account_id);

CREATE TABLE spent_confirms (
    token text PRIMARY KEY,
    used_at timestamptz NOT NULL
);
