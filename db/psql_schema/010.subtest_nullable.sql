BEGIN;

ALTER TABLE submission_tests DROP CONSTRAINT submission_tests_test_id_fkey;

ALTER TABLE submission_tests ALTER COLUMN test_id DROP NOT NULL;

ALTER TABLE submission_tests ADD CONSTRAINT submission_tests_test_id_fkey FOREIGN KEY (test_id) REFERENCES tests(id) ON DELETE SET NULL;

COMMIT;
