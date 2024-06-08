
-- Just "gravatar" and "discord" for now. Invalid values are treated as "gravatar"
ALTER TABLE users ADD COLUMN avatar_type TEXT NOT NULL DEFAULT 'gravatar';
