ALTER TABLE submission_tests ADD COLUMN contest_id bigint REFERENCES contests(id) ON DELETE SET NULL;
UPDATE submission_tests SET contest_id = subs.contest_id FROM submissions subs WHERE subs.id = submission_tests.submission_id;

ALTER TABLE submission_subtasks ADD COLUMN contest_id bigint REFERENCES contests(id) ON DELETE SET NULL;
UPDATE submission_subtasks SET contest_id = subs.contest_id FROM submissions subs WHERE subs.id = submission_subtasks.submission_id;


