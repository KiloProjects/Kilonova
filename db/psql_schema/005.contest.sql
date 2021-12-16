CREATE TABLE IF NOT EXISTS contests (
	id 				bigserial 	PRIMARY KEY,
	created_at 		timestamptz NOT NULL DEFAULT NOW(),

	start_date 		timestamptz NOT NULL DEFAULT -- TODO,
	duration 		interval 	NOT NULL DEFAULT 03:00:00, -- 3 hours
	
	pblist_id 		bigint 		NOT NULL REFERENCES problem_lists(id) ON DELETE RESTRICT,
);
