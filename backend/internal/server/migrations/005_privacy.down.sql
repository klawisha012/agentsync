ALTER TABLE accounts
    DROP COLUMN IF EXISTS share_copy,
    DROP COLUMN IF EXISTS share_view,
    DROP COLUMN IF EXISTS share_versions;
