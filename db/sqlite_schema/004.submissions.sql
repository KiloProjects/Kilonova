CREATE TABLE IF NOT EXISTS submissions (
	id 			INTEGER 	PRIMARY KEY,
	created_at 	TIMESTAMP 	NOT NULL DEFAULT CURRENT_TIMESTAMP,
	user_id 	INTEGER 	NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	problem_id 	INTEGER 	NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	language 	TEXT 		NOT NULL,
	code 		TEXT 		NOT NULL,
	status 		TEXT CHECK(status IN ('creating', 'waiting', 'working', 'finished')) NOT NULL DEFAULT 'creating',
	
	compile_error INTEGER,
	compile_message TEXT,

	score 		INTEGER 	NOT NULL DEFAULT 0,
	visible 	INTEGER 	NOT NULL DEFAULT FALSE,
	quality 	INTEGER 	NOT NULL DEFAULT FALSE
);
