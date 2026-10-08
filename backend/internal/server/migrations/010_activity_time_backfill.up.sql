UPDATE page_views
SET seen_at = CURRENT_TIMESTAMP
WHERE seen_at IS NULL;

UPDATE publication_likes
SET created_at = CURRENT_TIMESTAMP
WHERE created_at IS NULL;
