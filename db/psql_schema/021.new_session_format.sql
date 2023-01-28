
ALTER TABLE sessions ADD COLUMN expires_at timestamptz;
UPDATE sessions SET expires_at = created_at + INTERVAL '30 day';
ALTER TABLE sessions ALTER COLUMN expires_at SET NOT NULL;