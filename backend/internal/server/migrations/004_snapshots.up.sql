CREATE TABLE snapshots (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    agent text NOT NULL,
    number integer NOT NULL,
    packed bytea NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT snapshots_number_unique UNIQUE (account_id, agent, number)
);

CREATE TABLE snapshot_marks (
    account_id text NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    agent text NOT NULL,
    snapshot_id text NOT NULL REFERENCES snapshots (id) ON DELETE CASCADE,
    PRIMARY KEY (account_id, agent)
);
