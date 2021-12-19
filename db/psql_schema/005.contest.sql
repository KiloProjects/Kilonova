CREATE TABLE IF NOT EXISTS contests (
	id 				bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	created_at 		timestamptz NOT NULL DEFAULT NOW(),

	start_date 		timestamptz NOT NULL DEFAULT -- TODO,
	duration 		interval 	NOT NULL DEFAULT 03:00:00, -- 3 hours
	
	pblist_id 		bigint 		NOT NULL REFERENCES problem_lists(id) ON DELETE RESTRICT,
);

CREATE TABLE IF NOT EXISTS contest_registrations (
	user_id 	bigint 	NOT NULL 	REFERENCES users(id) ON DELETE CASCADE,
	contest_id 	bigint	NOT NULL 	REFERENCES contests(id) ON DELETE CASCADE,
	registered_at 	timestamptz 	NOT NULL DEFAULT NOW()
);

