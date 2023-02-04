
-- pblists

CREATE INDEX IF NOT EXISTS pblist_problems_index ON problem_list_problems (pblist_id);
CREATE INDEX IF NOT EXISTS pblist_pblists_index ON problem_list_pblists (parent_id);

-- submissions

CREATE INDEX IF NOT EXISTS problem_user_submissions_index ON submissions (user_id, problem_id);
CREATE INDEX IF NOT EXISTS problem_submissions_index ON submissions (problem_id);
CREATE INDEX IF NOT EXISTS contest_submissions_index ON submissions (contest_id);

CREATE INDEX IF NOT EXISTS submission_subtests_index ON submission_tests (submission_id);
CREATE INDEX IF NOT EXISTS submission_subtasks_index ON submission_subtasks (submission_id);

-- problems

CREATE INDEX IF NOT EXISTS problem_access_index ON problem_user_access (problem_id);
CREATE INDEX IF NOT EXISTS problem_attachments_index ON attachments (problem_id);
CREATE INDEX IF NOT EXISTS problem_tests_index ON tests (problem_id);

-- contests

CREATE INDEX IF NOT EXISTS contest_access_index ON contest_user_access (contest_id);
CREATE INDEX IF NOT EXISTS contest_problems_index ON contest_problems (contest_id);
CREATE INDEX IF NOT EXISTS contest_registrations_index ON contest_registrations (contest_id);
CREATE INDEX IF NOT EXISTS contest_questions_index ON contest_questions (contest_id);
CREATE INDEX IF NOT EXISTS contest_announcements_index ON contest_announcements (contest_id);

