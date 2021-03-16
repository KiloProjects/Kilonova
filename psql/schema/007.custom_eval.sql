
CREATE TYPE problem_type AS ENUM (
	'classic',
	'interactive',
	'custom_checker'
);

ALTER TYPE status ADD VALUE 'creating' BEFORE 'waiting';

ALTER TABLE problems ADD COLUMN pb_type problem_type NOT NULL DEFAULT 'classic';
ALTER TABLE problems ADD COLUMN helper_code text NOT NULL DEFAULT '';
ALTER TABLE problems ADD COLUMN helper_code_lang text NOT NULL DEFAULT 'cpp';

ALTER TABLE problems ADD COLUMN subtasks text NOT NULL DEFAULT '';
ALTER TABLE submission_tests ADD COLUMN skipped boolean NOT NULL DEFAULT false;
