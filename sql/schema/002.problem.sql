CREATE TABLE problems (
	id 			  bigserial 		PRIMARY KEY,
	created_at 	  timestamp 		NOT NULL DEFAULT NOW(),
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
	created_at 	timestamp 		NOT NULL DEFAULT NOW(),
	score 		integer 		NOT NULL,
	problem_id  bigint			NOT NULL REFERENCES problems(id),
	visible_id  bigint 			NOT NULL
);
