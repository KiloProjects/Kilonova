
ALTER TABLE submissions ALTER COLUMN status SET DEFAULT 'creating';
ALTER TABLE submissions ADD COLUMN quality BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE problems DROP COLUMN subtasks;
ALTER TABLE submission_tests DROP COLUMN skipped;

CREATE TABLE problem_lists (
	id 			bigserial 	PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	author_id 	bigint 		NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	title 		text 		NOT NULL DEFAULT '',
	description text 		NOT NULL DEFAULT '',
	list 		text 		NOT NULL DEFAULT ''
);

CREATE TABLE subtasks (
	id 			bigserial  	PRIMARY KEY,
	created_at  timestamptz NOT NULL DEFAULT NOW(),
	problem_id 	bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id 	integer 	NOT NULL,
	score 		integer 	NOT NULL,
	tests 		text 		NOT NULL
);
