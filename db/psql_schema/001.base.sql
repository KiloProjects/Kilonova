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
	
	verified_email 		boolean		NOT NULL DEFAULT false,
	email_verif_sent_at timestamptz,

	preferred_language text NOT NULL DEFAULT 'ro'
);

-- problem stuff

CREATE TABLE IF NOT EXISTS problems (
	id 			  	bigserial 			PRIMARY KEY,
	created_at 	  	timestamptz 		NOT NULL DEFAULT NOW(),
	name 		  	text 	    		NOT NULL,
	description   	text				NOT NULL DEFAULT '',
	test_name 	  	text      			NOT NULL,
	author_id 	  	bigint 				NOT NULL DEFAULT 1 REFERENCES users(id) ON DELETE SET DEFAULT,
	time_limit 	  	double precision 	NOT NULL DEFAULT 0.1,
	memory_limit  	integer   			NOT NULL DEFAULT 65536,

	source_size   	integer   			NOT NULL DEFAULT 10000,
	console_input 	boolean 			NOT NULL DEFAULT false,
	visible 	  	boolean 			NOT NULL DEFAULT false,

	source_credits 	text				NOT NULL DEFAULT '',
	author_credits 	text				NOT NULL DEFAULT '',
	short_description text				NOT NULL DEFAULT '',
	default_points 	integer 			NOT NULL DEFAULT 0
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

CREATE TYPE status AS ENUM (
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
	compile_error 	boolean,
	compile_message text,
	score 			integer			NOT NULL DEFAULT 0,
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
	description text 		NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS problem_list_problems (
    pblist_id bigint NOT NULL REFERENCES problem_lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    problem_id bigint NOT NULL REFERENCES problems(id) ON DELETE CASCADE ON UPDATE CASCADE,
    position bigint NOT NULL DEFAULT 0,
    UNIQUE (pblist_id, problem_id)
);

CREATE TABLE IF NOT EXISTS subtasks (
	id 			bigserial  	PRIMARY KEY,
	created_at  timestamptz NOT NULL DEFAULT NOW(),
	problem_id 	bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id 	integer 	NOT NULL,
	score 		integer 	NOT NULL
);

CREATE TABLE IF NOT EXISTS subtask_tests (
    subtask_id bigint NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    test_id bigint NOT NULL REFERENCES tests(id) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE (subtask_id, test_id)
);

CREATE TABLE IF NOT EXISTS sessions (
	id 			text 		PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	user_id 	integer 	NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS verifications (
	id 			text 		PRIMARY KEY,
	created_at 	timestamptz NOT NULL DEFAULT NOW(),
	user_id 	integer 	NOT NULL REFERENCES users(id) ON DELETE CASCADE
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

CREATE TABLE IF NOT EXISTS audit_logs (
    id          bigserial 	    PRIMARY KEY,
    logged_at   timestamptz     NOT NULL DEFAULT NOW(),
    system_log  boolean         NOT NULL DEFAULT false,
    msg         text            NOT NULL DEFAULT '',
    author_id   bigint          DEFAULT null REFERENCES users(id) ON DELETE SET NULL
);
