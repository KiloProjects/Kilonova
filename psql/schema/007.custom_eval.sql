
CREATE TYPE problem_type AS ENUM (
	'classic',
	'interactive',
	'custom_checker'
);

ALTER TABLE problems ADD COLUMN pb_type problem_type NOT NULL DEFAULT 'classic';
ALTER TABLE problems ADD COLUMN helper_code text NOT NULL DEFAULT '';
ALTER TABLE problems ADD COLUMN helper_code_lang text NOT NULL DEFAULT '';
