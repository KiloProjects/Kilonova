CREATE TABLE users (
	id 			bigserial 	PRIMARY KEY,
	created_at	timestamp 	NOT NULL DEFAULT NOW(),
	name 		text 	  	NOT NULL UNIQUE,
	admin 		boolean 	NOT NULL DEFAULT false,
	proposer 	boolean		NOT NULL DEFAULT false,
	email 		text 	  	NOT NULL UNIQUE,
	password 	text 	  	NOT NULL
);

