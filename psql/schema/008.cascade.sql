ALTER TABLE submissions DROP CONSTRAINT submissions_user_id_fkey;
ALTER TABLE submissions ADD CONSTRAINT submissions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE submission_tests DROP CONSTRAINT submission_tests_user_id_fkey;
ALTER TABLE submission_tests ADD CONSTRAINT submission_tests_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE problems ALTER COLUMN author_id SET DEFAULT 1;
ALTER TABLE problems DROP CONSTRAINT problems_author_id_fkey;
ALTER TABLE problems ADD CONSTRAINT problems_author_id_fkey FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET DEFAULT;

