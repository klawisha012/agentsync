CREATE TABLE page_views (
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    viewer_key text NOT NULL,
    PRIMARY KEY (account_id, viewer_key)
);

CREATE TABLE page_likes (
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    liker_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, liker_id)
);

CREATE TABLE machines (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    host text NOT NULL,
    listener text NOT NULL,
    token text NOT NULL,
    CONSTRAINT machines_token_unique UNIQUE (token)
);

CREATE TABLE publications (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    agent text NOT NULL,
    version integer NOT NULL,
    packed bytea NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT publications_version_unique UNIQUE (account_id, agent, version)
);
