ALTER TABLE users ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE (current_setting('TIMEZONE'));
ALTER TABLE problems ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE (current_setting('TIMEZONE'));
ALTER TABLE tests ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE (current_setting('TIMEZONE'));
ALTER TABLE submissions ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE (current_setting('TIMEZONE'));
ALTER TABLE submission_tests ALTER created_at TYPE timestamptz USING created_at AT TIME ZONE (current_setting('TIMEZONE'));
