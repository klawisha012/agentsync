ALTER TABLE page_views
    DROP COLUMN IF EXISTS seen_at;

ALTER TABLE publication_likes
    DROP COLUMN IF EXISTS created_at;
