

CREATE TYPE scoring_type AS enum (
    'max_submission',
    'sum_subtasks'
);

ALTER TABLE problems ADD COLUMN scoring_strategy scoring_type NOT NULL DEFAULT 'max_submission';

CREATE OR REPLACE VIEW submission_subtask_max_scores (submission_id, problem_id, user_id, subtask_id, max_score) AS
    SELECT DISTINCT ON (stks.subtask_id) first_value(stks.submission_id) OVER stk_id_window AS submission_id, stks.problem_id, stks.user_id, stks.subtask_id, MAX(ROUND(ljl.pcg * stks.score / 100.0)) OVER stk_id_window AS max_score
        FROM submission_subtasks stks 
            LEFT JOIN LATERAL (SELECT MIN(sts.score) AS pcg FROM submission_subtask_subtests ssst, submission_tests sts 
            WHERE stks.id = ssst.submission_subtask_id AND sts.id = ssst.submission_test_id) ljl ON TRUE
        WHERE stks.subtask_id IS NOT NULL
        WINDOW stk_id_window AS (PARTITION BY stks.subtask_id ORDER BY ljl.pcg DESC);

-- TODO: also have submission time??
CREATE OR REPLACE VIEW max_score_view (user_id, problem_id, score) AS 
    SELECT users.id user_id, 
            pbs.id problem_id, 
            CASE WHEN pbs.scoring_strategy = 'max_submission' THEN ms_sub.max_score
                 WHEN pbs.scoring_strategy = 'sum_subtasks'   THEN ms_subtask.max_score::integer
                 ELSE -1
            END score
    FROM (problems pbs CROSS JOIN users)
    LEFT JOIN LATERAL (SELECT coalesce(MAX(score), -1) AS max_score FROM submissions WHERE user_id = users.id AND problem_id = pbs.id) ms_sub ON TRUE
    LEFT JOIN LATERAL (SELECT coalesce(SUM(max_score), -1) AS max_score FROM submission_subtask_max_scores WHERE user_id = users.id AND problem_id = pbs.id) ms_subtask ON TRUE;

