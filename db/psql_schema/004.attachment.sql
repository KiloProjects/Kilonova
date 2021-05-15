CREATE TABLE IF NOT EXISTS attachments (
	id 			bigserial 	PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	problem_id 	bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible 	boolean 	NOT NULL DEFAULT false,

	name 		text 		NOT NULL,
	data 		bytea 		NOT NULL
);
