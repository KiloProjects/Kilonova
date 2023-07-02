
ALTER TABLE problems ADD COLUMN published_at timestamptz;
UPDATE problems SET published_at = NOW() WHERE visible = true;

ALTER TABLE blog_posts ADD COLUMN published_at timestamptz;
