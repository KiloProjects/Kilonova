-- Sessions created without "keep me signed in" are short-lived (24h, sliding).
-- Everything created before this migration was a 30-day session.
ALTER TABLE sessions ADD COLUMN persistent boolean NOT NULL DEFAULT true;

-- The view was created with SELECT *, which froze its column list.
CREATE OR REPLACE VIEW active_sessions AS SELECT * FROM sessions WHERE NOW() < expires_at;
