
ALTER TABLE problem_lists ADD COLUMN list_ids integer[] NOT NULL DEFAULT '{}';
UPDATE problem_lists SET list_ids = regexp_split_to_array(list, ',')::integer[];
ALTER TABLE problem_lists DROP COLUMN list;
ALTER TABLE problem_lists RENAME COLUMN list_ids TO list;

ALTER TABLE subtasks ADD COLUMN test_ids integer[] NOT NULL DEFAULT '{}';
UPDATE subtasks SET test_ids = regexp_split_to_array(tests, ',')::integer[];
ALTER TABLE subtasks DROP COLUMN tests;
ALTER TABLE subtasks RENAME COLUMN test_ids TO tests;


