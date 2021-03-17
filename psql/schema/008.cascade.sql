-- User references

ALTER TABLE submissions DROP CONSTRAINT submissions_user_id_fkey;
ALTER TABLE submissions ADD CONSTRAINT submissions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE submission_tests DROP CONSTRAINT submission_tests_user_id_fkey;
ALTER TABLE submission_tests ADD CONSTRAINT submission_tests_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE problems ALTER COLUMN author_id SET DEFAULT 1;
ALTER TABLE problems DROP CONSTRAINT problems_author_id_fkey;
ALTER TABLE problems ADD CONSTRAINT problems_author_id_fkey FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET DEFAULT;

-- Problem references

ALTER TABLE tests DROP CONSTRAINT tests_problem_id_fkey;
ALTER TABLE tests ADD CONSTRAINT tests_problem_id_fkey FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE;

ALTER TABLE submissions DROP CONSTRAINT submissions_problem_id_fkey;
ALTER TABLE submissions ADD CONSTRAINT submissions_problem_id_fkey FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE;

-- Test references
ALTER TABLE submission_tests DROP CONSTRAINT submission_tests_test_id_fkey;
ALTER TABLE submission_tests ADD CONSTRAINT submission_tests_test_id_fkey FOREIGN KEY (test_id) REFERENCES tests(id) ON DELETE CASCADE;

-- Submission references
ALTER TABLE submission_tests DROP CONSTRAINT submission_tests_submission_id_fkey;
ALTER TABLE submission_tests ADD CONSTRAINT submission_tests_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES submissions(id) ON DELETE CASCADE;
