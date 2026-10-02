DROP TABLE IF EXISTS account_avatars;

ALTER TABLE accounts DROP COLUMN IF EXISTS avatar_updated;
ALTER TABLE accounts DROP COLUMN IF EXISTS has_avatar;
