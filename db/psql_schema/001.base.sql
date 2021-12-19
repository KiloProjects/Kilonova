-- user stuff

CREATE TABLE IF NOT EXISTS users (
	id 					bigserial 	PRIMARY KEY,
	created_at			timestamptz	NOT NULL DEFAULT NOW(),
	name 				text 	  	NOT NULL UNIQUE,
	admin 				boolean 	NOT NULL DEFAULT false,
	proposer 			boolean		NOT NULL DEFAULT false,
	email 				text 	  	NOT NULL UNIQUE,
	password 			text 	  	NOT NULL,
	bio 				text 		NOT NULL DEFAULT '',
	
	default_visible 	boolean		NOT NULL DEFAULT false,

	verified_email 		boolean		NOT NULL DEFAULT false,
	email_verif_sent_at timestamptz,

	banned 				boolean 	NOT NULL DEFAULT false,
	disabled 			boolean 	NOT NULL DEFAULT false,
	preferred_language text NOT NULL DEFAULT 'ro'
);

-- problem stuff

CREATE TYPE IF NOT EXISTS problem_type AS ENUM (
	'classic',
	'interactive',
	'custom_checker'
);

CREATE TABLE IF NOT EXISTS problems (
	id 			  	bigserial 			PRIMARY KEY,
	created_at 	  	timestamptz 		NOT NULL DEFAULT NOW(),
	name 		  	text 	    		NOT NULL UNIQUE,
	description   	text				NOT NULL DEFAULT '',
	test_name 	  	text      			NOT NULL,
	author_id 	  	bigint 				NOT NULL DEFAULT 1 REFERENCES users(id) ON DELETE SET DEFAULT,
	time_limit 	  	double precision 	NOT NULL DEFAULT 0.1,
	memory_limit  	integer   			NOT NULL DEFAULT 65536,
	stack_limit   	integer   			NOT NULL DEFAULT 16384,

	source_size   	integer   			NOT NULL DEFAULT 10000,
	console_input 	boolean 			NOT NULL DEFAULT false,
	visible 	  	boolean 			NOT NULL DEFAULT false,

	source_credits 	text				NOT NULL DEFAULT '',
	author_credits 	text				NOT NULL DEFAULT '',
	short_description text				NOT NULL DEFAULT '',
	default_points 	integer 			NOT NULL DEFAULT 0,

	pb_type 		problem_type 		NOT NULL DEFAULT 'classic',
	helper_code 	text 				NOT NULL DEFAULT '',
	helper_code_lang text 				NOT NULL DEFAULT 'cpp',
);

CREATE TABLE IF NOT EXISTS tests (
	id 			bigserial  		PRIMARY KEY,
	created_at 	timestamptz		NOT NULL DEFAULT NOW(),
	score 		integer 		NOT NULL,
	problem_id  bigint			NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id  bigint 			NOT NULL,
	orphaned 	bool 			NOT NULL DEFAULT false
);

-- submissions stuff

CREATE TYPE IF NOT EXISTS status AS ENUM (
	'creating',
	'waiting',
	'working',
	'finished'
);

CREATE TABLE IF NOT EXISTS submissions (
	id 				bigserial 		PRIMARY KEY,
	created_at 		timestamptz		NOT NULL DEFAULT NOW(),
	user_id 		bigint			NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	problem_id 		bigint  		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	language		text 			NOT NULL,
	code 			text 			NOT NULL,
	status 			status 			NOT NULL DEFAULT 'creating',
	quality 		BOOLEAN 		NOT NULL DEFAULT false
	compile_error 	boolean,
	compile_message text,
	score 			integer			NOT NULL DEFAULT 0,
	visible 		boolean			NOT NULL DEFAULT false,
	max_time 		DOUBLE PRECISION NOT NULL DEFAULT -1,
	max_memory 		INTEGER 		NOT NULL DEFAULT -1
);

CREATE TABLE IF NOT EXISTS submission_tests (
	id 				bigserial			PRIMARY KEY,
	created_at 		timestamptz			NOT NULL DEFAULT NOW(),
	done			boolean 			NOT NULL DEFAULT false,
	verdict 		varchar(255) 		NOT NULL DEFAULT '',
	time 			double precision	NOT NULL DEFAULT 0,
	memory 			integer				NOT NULL DEFAULT 0,
	score 			integer				NOT NULL DEFAULT 0,
	test_id			bigint  			NOT NULL REFERENCES tests(id) ON DELETE CASCADE,
	user_id 		bigint  			NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	submission_id 	bigint  			NOT NULL REFERENCES submissions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS problem_lists (
	id 			bigserial 	PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	author_id 	bigint 		NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	title 		text 		NOT NULL DEFAULT '',
	description text 		NOT NULL DEFAULT '',
	list 		text 		NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS subtasks (
	id 			bigserial  	PRIMARY KEY,
	created_at  timestamptz NOT NULL DEFAULT NOW(),
	problem_id 	bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id 	integer 	NOT NULL,
	score 		integer 	NOT NULL,
	tests 		text 		NOT NULL
);

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

CREATE TABLE IF NOT EXISTS attachments (
	id 			bigserial 	PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	problem_id 	bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible 	boolean 	NOT NULL DEFAULT true,
	private 	boolean 	NOT NULL DEFAULT false,

	name 		text 		NOT NULL,
	data 		bytea 		NOT NULL,
	data_size 	INTEGER 	GENERATED ALWAYS AS (length(data)) STORED
);

--
--CREATE TABLE IF NOT EXISTS test_inputs (
--	tid 	bigint 	NOT NULL UNIQUE REFERENCES tests(id),
--	data 	bytea 	NOT NULL
--);

--CREATE TABLE IF NOT EXISTS test_outputs (
--	tid 	bigint 	NOT NULL UNIQUE REFERENCES tests(id),
--	data 	bytea 	NOT NULL
--);

--CREATE TABLE IF NOT EXISTS subtest_outputs (
--	stid 	bigint 	NOT NULL UNIQUE REFERENCES submission_tests(id),
--	data 	bytea 	NOT NULL
--);


