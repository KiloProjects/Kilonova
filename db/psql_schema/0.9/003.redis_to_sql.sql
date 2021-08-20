CREATE TABLE IF NOT EXISTS sessions (
	id 			text 		PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	user_id 	integer 	NOT NULL REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS verifications (
	id 			text 		PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	user_id 	integer 	NOT NULL REFERENCES users(id)
);
