ALTER TABLE accounts
    ADD COLUMN has_avatar boolean NOT NULL DEFAULT false,
    ADD COLUMN avatar_updated timestamptz;

CREATE TABLE account_avatars (
    account_id text PRIMARY KEY REFERENCES accounts (id) ON DELETE CASCADE,
    media_type text NOT NULL,
    body bytea NOT NULL
);
