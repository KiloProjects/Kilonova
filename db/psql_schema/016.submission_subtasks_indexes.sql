-- contest_max_scores filters submission_subtasks by contest_id; without this every leaderboard read seq-scans the table.
CREATE INDEX IF NOT EXISTS submission_subtasks_contest_index ON submission_subtasks (contest_id);

-- Declared in 001.base.sql but missing on databases created before they were added there.
CREATE INDEX IF NOT EXISTS submission_subtasks_score_index ON submission_subtasks (user_id, problem_id);
CREATE INDEX IF NOT EXISTS submission_subtasks_problem_index ON submission_subtasks (problem_id);
