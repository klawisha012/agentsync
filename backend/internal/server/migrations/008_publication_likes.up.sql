CREATE TABLE publication_likes (
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    agent_key text NOT NULL,
    liker_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, agent_key, liker_id)
);
