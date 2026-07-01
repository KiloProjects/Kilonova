
ALTER TABLE problems ADD COLUMN review_requested_at timestamptz;
ALTER TABLE problems ADD COLUMN review_requested_by bigint REFERENCES users(id) ON DELETE SET NULL;