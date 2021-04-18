CREATE TABLE IF NOT EXISTS problems (
	id 			INTEGER 	PRIMARY KEY,
	created_at 	TIMESTAMP 	NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name 		TEXT 		NOT NULL UNIQUE,
	description TEXT 		NOT NULL DEFAULT '',
	test_name 	TEXT 		NOT NULL,
	author_id 	INTEGER 	NOT NULL DEFAULT 1 REFERENCES users(id) ON DELETE SET DEFAULT,
	time_limit 	FLOAT 		NOT NULL DEFAULT 0.1,
	memory_limit INTEGER 	NOT NULL DEFAULT 65536,
	stack_limit INTEGER 	NOT NULL DEFAULT 16384,

	source_size INTEGER 	NOT NULL DEFAULT 10000,
	console_input INTEGER 	NOT NULL DEFAULT FALSE,
	visible 	INTEGER 	NOT NULL DEFAULT FALSE,

	source_credits TEXT 	NOT NULL DEFAULT '',
	author_credits TEXT 	NOT NULL DEFAULT '',
	short_description TEXT 	NOT NULL DEFAULT '',
	default_points INTEGER 	NOT NULL DEFAULT 0,

	pb_type 	TEXT CHECK(pb_type IN ('classic', 'interactive', 'custom_checker')) NOT NULL DEFAULT 'classic',
	helper_code TEXT 		NOT NULL DEFAULT '',
	helper_code_lang TEXT 	NOT NULL DEFAULT 'cpp'
);
