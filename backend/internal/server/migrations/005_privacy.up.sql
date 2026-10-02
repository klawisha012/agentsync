ALTER TABLE accounts
    ADD COLUMN share_copy boolean NOT NULL DEFAULT false,
    ADD COLUMN share_view boolean NOT NULL DEFAULT false,
    ADD COLUMN share_versions boolean NOT NULL DEFAULT false;
