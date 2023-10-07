BEGIN DEFERRABLE;

DROP VIEW IF EXISTS max_score_view CASCADE;
DROP VIEW IF EXISTS max_score_contest_view CASCADE;
DROP VIEW IF EXISTS submission_subtask_max_scores CASCADE;
DROP VIEW IF EXISTS contest_submission_subtask_max_scores CASCADE;


ALTER TABLE problems ADD COLUMN digit_precision integer NOT NULL DEFAULT 0;

ALTER TABLE problems ALTER COLUMN default_points TYPE numeric;
ALTER TABLE tests ALTER COLUMN score TYPE numeric;
ALTER TABLE subtasks ALTER COLUMN score TYPE numeric;

ALTER TABLE submissions ADD COLUMN digit_precision integer NOT NULL DEFAULT 0;
ALTER TABLE submissions ALTER COLUMN score TYPE numeric;
UPDATE submissions SET digit_precision = COALESCE((SELECT digit_precision FROM problems WHERE problems.id = problem_id), 0);

ALTER TABLE submission_tests ALTER COLUMN score TYPE numeric;
ALTER TABLE submission_tests RENAME COLUMN score TO percentage;

ALTER TABLE submission_tests ALTER COLUMN max_score TYPE numeric;
ALTER TABLE submission_tests RENAME COLUMN max_score TO score;

ALTER TABLE submission_subtasks ALTER COLUMN final_percentage TYPE numeric;
ALTER TABLE submission_subtasks ALTER COLUMN score TYPE numeric;
ALTER TABLE submission_subtasks ADD COLUMN digit_precision integer NOT NULL DEFAULT 0;
UPDATE submission_subtasks SET digit_precision = COALESCE((SELECT digit_precision FROM submissions WHERE submissions.id = submission_id), 0);

COMMIT;
-- don't forget to rerun 002.views.sql here
