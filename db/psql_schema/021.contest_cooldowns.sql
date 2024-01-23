
ALTER TABLE contests ADD COLUMN question_cooldown_ms integer NOT NULL DEFAULT 2000; -- 2s
ALTER TABLE contests ADD COLUMN submission_cooldown_ms integer NOT NULL DEFAULT 10000; -- 10s

