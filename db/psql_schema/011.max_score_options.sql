

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
    WITH min_percentage AS (
        SELECT ssst.submission_subtask_id AS sub_subtask_id, MIN(sts.score) AS pcg FROM submission_subtask_subtests ssst, submission_tests sts 
            WHERE sts.id = ssst.submission_test_id GROUP BY ssst.submission_subtask_id
    ) SELECT MAX(stks.problem_id) as problem_id, stks.user_id, stks.subtask_id, MAX(ljl.pcg) AS pcg, MAX(ROUND(COALESCE(ljl.pcg, 0) * stks.score / 100.0)) max_score
        FROM submission_subtasks stks 
            LEFT JOIN min_percentage ljl ON ljl.sub_subtask_id = stks.subtask_id
        WHERE stks.subtask_id IS NOT NULL GROUP BY stks.user_id, stks.subtask_id;

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
    -- LEFT JOIN LATERAL (SELECT coalesce(MAX(score), -1) AS max_score FROM submissions WHERE user_id = users.id AND problem_id = pbs.id) ms_sub ON TRUE
    -- LEFT JOIN LATERAL (SELECT coalesce(SUM(max_score), -1) AS max_score FROM submission_subtask_max_scores WHERE user_id = users.id AND problem_id = pbs.id) ms_subtask ON TRUE;

