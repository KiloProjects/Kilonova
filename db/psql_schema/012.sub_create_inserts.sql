-- These can be used to insert stuff 
-- Insert submission
INSERT INTO submissions (user_id, problem_id, contest_id, language, code) VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- Insert submission tests
INSERT INTO submission_tests (user_id, submission_id, test_id, visible_id, max_score) SELECT $1, $2, id, visible_id, score FROM tests WHERE problem_id = $3 AND orphaned = false;

-- Insert submission subtasks
INSERT INTO submission_subtasks (user_id, submission_id, subtask_id, problem_id, visible_id, score) SELECT $1, $2, id, problem_id, visible_id, score FROM subtasks WHERE problem_id = $3;
INSERT INTO submission_subtask_subtests 
       SELECT sstk.id, st.id
       FROM subtask_tests stks
       INNER JOIN submission_subtasks sstk ON stks.subtask_id = sstk.subtask_id
       INNER JOIN submission_tests st ON stks.test_id = st.test_id AND sstk.submission_id = st.submission_id WHERE st.submission_id = $1;