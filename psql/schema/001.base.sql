-- user stuff

CREATE TABLE users (
	id 			bigserial 		PRIMARY KEY,
	created_at	timestamptz		NOT NULL DEFAULT NOW(),
	name 		text 	  		NOT NULL UNIQUE,
	admin 		boolean 		NOT NULL DEFAULT false,
	proposer 	boolean			NOT NULL DEFAULT false,
	email 		text 	  		NOT NULL UNIQUE,
	password 	text 	  		NOT NULL,
	bio 		text 			NOT NULL DEFAULT '',
	
	default_visible bool 		NOT NULL DEFAULT false
);

-- problem stuff

CREATE TABLE problems (
	id 			  bigserial 		PRIMARY KEY,
	created_at 	  timestamptz 		NOT NULL DEFAULT NOW(),
	name 		  text 	    		NOT NULL UNIQUE,
	description   text				NOT NULL DEFAULT '',
	test_name 	  text      		NOT NULL,
	author_id 	  bigint 			NOT NULL REFERENCES users(id),
	time_limit 	  double precision 	NOT NULL DEFAULT 0.1,
	memory_limit  integer   		NOT NULL DEFAULT 65536,
	stack_limit   integer   		NOT NULL DEFAULT 16384,

	source_size   integer   		NOT NULL DEFAULT 10000,
	console_input boolean 			NOT NULL DEFAULT false,
	visible 	  boolean 			NOT NULL DEFAULT false
);

CREATE TABLE tests (
	id 			bigserial  		PRIMARY KEY,
	created_at 	timestamptz		NOT NULL DEFAULT NOW(),
	score 		integer 		NOT NULL,
	problem_id  bigint			NOT NULL REFERENCES problems(id),
	visible_id  bigint 			NOT NULL,
	orphaned 	bool 			NOT NULL DEFAULT false
);

-- submissions stuff

CREATE TYPE status AS ENUM (
	'waiting',
	'working',
	'finished'
);

CREATE TABLE submissions (
	id 				bigserial 		PRIMARY KEY,
	created_at 		timestamptz		NOT NULL DEFAULT NOW(),
	user_id 		bigint			NOT NULL REFERENCES users(id),
	problem_id 		bigint  		NOT NULL REFERENCES problems(id),
	language		text 			NOT NULL,
	code 			text 			NOT NULL,
	status 			status 			NOT NULL DEFAULT 'waiting',
	compile_error 	boolean,
	compile_message text,
	score 			integer			NOT NULL DEFAULT 0,
	visible 		boolean			NOT NULL DEFAULT false
);

CREATE TABLE submission_tests (
	id 				bigserial			PRIMARY KEY,
	created_at 		timestamptz			NOT NULL DEFAULT NOW(),
	done			boolean 			NOT NULL DEFAULT false,
	verdict 		varchar(255) 		NOT NULL DEFAULT '',
	time 			double precision	NOT NULL DEFAULT 0,
	memory 			integer				NOT NULL DEFAULT 0,
	score 			integer				NOT NULL DEFAULT 0,
	test_id			bigint  			NOT NULL REFERENCES tests(id),
	user_id 		bigint  			NOT NULL REFERENCES users(id),
	submission_id 	bigint  			NOT NULL REFERENCES submissions(id)
);
