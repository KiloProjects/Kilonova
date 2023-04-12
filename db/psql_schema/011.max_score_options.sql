

ALTER TABLE submission_subtasks ADD COLUMN final_percentage integer DEFAULT NULL;

WITH min_percentage AS (
    SELECT ssst.submission_subtask_id AS sub_subtask_id, MIN(sts.score) AS pcg FROM submission_subtask_subtests ssst, submission_tests sts 
        WHERE sts.id = ssst.submission_test_id GROUP BY ssst.submission_subtask_id
) UPDATE submission_subtasks 
    SET final_percentage = min_percentage.pcg 
    FROM min_percentage WHERE min_percentage.sub_subtask_id = submission_subtasks.id

CREATE TYPE scoring_type AS enum (
    'max_submission',
    'sum_subtasks'
);

ALTER TABLE problems ADD COLUMN scoring_strategy scoring_type NOT NULL DEFAULT 'max_submission';

CREATE INDEX IF NOT EXISTS submission_subtests_id_index ON submission_tests (id);
CREATE INDEX IF NOT EXISTS submission_subtasks_score_index ON submission_subtasks (user_id, problem_id);

CREATE OR REPLACE VIEW submission_subtask_max_scores (problem_id, user_id, subtask_id, max_score) AS
    SELECT MAX(problem_id) as problem_id, user_id, subtask_id, MAX(ROUND(COALESCE(final_percentage, 0) * score / 100.0)) max_score
    FROM submission_subtasks stks
    WHERE subtask_id IS NOT NULL GROUP BY user_id, subtask_id;

CREATE OR REPLACE VIEW max_score_view (user_id, problem_id, score) AS 
    WITH max_submission_strat AS (
        SELECT user_id, problem_id, MAX(score) AS max_score FROM submissions GROUP BY user_id, problem_id
    ), sum_subtasks_strat AS (
        SELECT user_id, problem_id, coalesce(SUM(max_score), -1) AS max_score FROM submission_subtask_max_scores GROUP BY user_id, problem_id
    ) SELECT users.id user_id, 
            pbs.id problem_id, 
            CASE WHEN pbs.scoring_strategy = 'max_submission' THEN COALESCE(ms_sub.max_score, -1)
                 WHEN pbs.scoring_strategy = 'sum_subtasks'   THEN COALESCE(ms_subtask.max_score::integer, -1)
                 ELSE -1
            END score
    FROM (problems pbs CROSS JOIN users) 
        LEFT JOIN max_submission_strat ms_sub ON (ms_sub.user_id = users.id AND ms_sub.problem_id = pbs.id)
        LEFT JOIN sum_subtasks_strat ms_subtask ON (ms_subtask.user_id = users.id AND ms_subtask.problem_id = pbs.id);

CREATE OR REPLACE VIEW contest_submission_subtask_max_scores (problem_id, user_id, subtask_id, contest_id, max_score) AS
    SELECT MAX(stks.problem_id) as problem_id, stks.user_id, subtask_id, subs.contest_id, MAX(ROUND(COALESCE(final_percentage, 0) * stks.score / 100.0)) max_score
    FROM submission_subtasks stks INNER JOIN submissions subs ON stks.submission_id = subs.id
    WHERE subtask_id IS NOT NULL AND subs.contest_id IS NOT NULL GROUP BY stks.user_id, subtask_id, subs.contest_id;

-- TODO: Refactor this like above
CREATE OR REPLACE VIEW max_score_contest_view (user_id, problem_id, contest_id, score) AS 
    WITH max_submission_strat AS (
        SELECT user_id, problem_id, contest_id, MAX(score) AS max_score FROM submissions WHERE contest_id IS NOT NULL GROUP BY user_id, problem_id, contest_id
    ), sum_subtasks_strat AS (
        SELECT user_id, problem_id, contest_id, coalesce(SUM(max_score), -1) AS max_score FROM contest_submission_subtask_max_scores GROUP BY user_id, problem_id, contest_id
    ) SELECT users.user_id user_id, 
            pbs.problem_id problem_id,
            users.contest_id contest_id, 
            CASE WHEN problems.scoring_strategy = 'max_submission' THEN COALESCE(ms_sub.max_score, -1)
                 WHEN problems.scoring_strategy = 'sum_subtasks'   THEN COALESCE(ms_subtask.max_score::integer, -1)
                 ELSE -1
            END score
    FROM ((contest_problems pbs INNER JOIN contest_registrations users ON users.contest_id = pbs.contest_id) INNER JOIN problems ON pbs.problem_id = problems.id) 
        LEFT JOIN max_submission_strat ms_sub ON (ms_sub.user_id = users.user_id AND ms_sub.problem_id = pbs.problem_id AND ms_sub.contest_id = users.contest_id)
        LEFT JOIN sum_subtasks_strat ms_subtask ON (ms_subtask.user_id = users.user_id AND ms_subtask.problem_id = pbs.problem_id AND ms_subtask.contest_id = users.contest_id);

-- Since we now return -1 on no attempt, we must filter it when computing the top view
CREATE OR REPLACE VIEW contest_top_view
    AS WITH contest_scores AS (
        SELECT user_id, contest_id, SUM(score) AS total_score FROM max_score_contest_view WHERE score >= 0 GROUP BY user_id, contest_id
    ) SELECT users.user_id, users.contest_id, COALESCE(scores.total_score, 0) AS total_score 
    FROM contest_registrations users LEFT OUTER JOIN contest_scores scores ON users.user_id = scores.user_id AND users.contest_id = scores.contest_id ORDER BY contest_id, total_score DESC, user_id;



