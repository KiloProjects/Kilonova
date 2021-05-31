
CREATE TABLE IF NOT EXISTS test_inputs (
	tid 	bigint 	NOT NULL UNIQUE REFERENCES tests(id),
	data 	bytea 	NOT NULL
);

CREATE TABLE IF NOT EXISTS test_outputs (
	tid 	bigint 	NOT NULL UNIQUE REFERENCES tests(id),
	data 	bytea 	NOT NULL
);

CREATE TABLE IF NOT EXISTS subtest_outputs (
	stid 	bigint 	NOT NULL UNIQUE REFERENCES submission_tests(id),
	data 	bytea 	NOT NULL
);

