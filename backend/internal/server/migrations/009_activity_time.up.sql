ALTER TABLE page_views
    ADD COLUMN seen_at timestamptz;

ALTER TABLE publication_likes
    ADD COLUMN created_at timestamptz;
