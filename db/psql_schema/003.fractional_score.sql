BEGIN DEFERRABLE;

DROP VIEW max_score_view CASCADE;
DROP VIEW max_score_contest_view CASCADE;
DROP VIEW submission_subtask_max_scores CASCADE;
DROP VIEW contest_submission_subtask_max_scores CASCADE;


ALTER TABLE problems ADD COLUMN digit_precision integer NOT NULL DEFAULT 0;

ALTER TABLE tests ALTER COLUMN score TYPE numeric;
ALTER TABLE subtasks ALTER COLUMN score TYPE numeric;

ALTER TABLE submissions ALTER COLUMN score TYPE numeric;

ALTER TABLE submission_tests ALTER COLUMN score TYPE numeric;
ALTER TABLE submission_tests RENAME COLUMN score TO percentage;

ALTER TABLE submission_tests ALTER COLUMN max_score TYPE numeric;
ALTER TABLE submission_tests RENAME COLUMN max_score TO score;

ALTER TABLE submission_subtasks ALTER COLUMN final_percentage TYPE numeric;
ALTER TABLE submission_subtasks ALTER COLUMN score TYPE numeric;

-- rerun 002.views.sql here


COMMIT;