ALTER TABLE accounts
    ADD COLUMN comment_access text NOT NULL DEFAULT 'hidden';

CREATE TABLE profile_comments (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    author_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    body text NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX profile_comments_account_idx ON profile_comments (account_id, created_at);
