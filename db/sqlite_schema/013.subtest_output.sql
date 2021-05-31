CREATE TABLE IF NOT EXISTS subtest_outputs (
	stid 	INTEGER 	NOT NULL UNIQUE REFERENCES submission_tests(id),
	data 	BLOB 		NOT NULL
);
