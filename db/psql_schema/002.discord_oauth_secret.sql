
CREATE TABLE IF NOT EXISTS discord_oauth_secrets (
    id          text        PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
    user_id     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

ALTER TABLE users ADD COLUMN discord_id text DEFAULT NULL;
