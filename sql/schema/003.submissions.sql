CREATE TYPE status AS ENUM (
	'waiting',
	'working',
	'finished'
);

CREATE TABLE submissions (
	id 				bigserial 	PRIMARY KEY,
	created_at 		timestamp 	NOT NULL DEFAULT NOW(),
	user_id 		bigint		NOT NULL REFERENCES users(id),
	problem_id 		bigint  	NOT NULL REFERENCES problems(id),
	language		text 		NOT NULL,
	code 			text 		NOT NULL,
	status 			status 		NOT NULL DEFAULT 'waiting',
	compile_error 	boolean,
	compile_message text,
	score 			integer		NOT NULL DEFAULT 0,
	visible 		boolean		NOT NULL DEFAULT false
);

CREATE TABLE submission_tests (
	id 				bigserial			PRIMARY KEY,
	created_at 		timestamp 			NOT NULL DEFAULT NOW(),
	done			boolean 			NOT NULL DEFAULT false,
	verdict 		varchar(255) 		NOT NULL DEFAULT '',
	time 			double precision	NOT NULL DEFAULT 0,
	memory 			integer				NOT NULL DEFAULT 0,
	score 			integer				NOT NULL DEFAULT 0,
	test_id			bigint  			NOT NULL REFERENCES tests(id),
	user_id 		bigint  			NOT NULL REFERENCES users(id),
	submission_id 	bigint  			NOT NULL REFERENCES submissions(id)
);
