BEGIN;


CREATE OR REPLACE FUNCTION max_score(user_id bigint, problem_id bigint, OUT mscore decimal) AS $$
DECLARE
    scoring_strat text;
BEGIN
    SELECT scoring_strategy INTO scoring_strat FROM problems WHERE id = $2 LIMIT 1;
    IF scoring_strat = 'max_submission' OR scoring_strat = 'acm-icpc' THEN
        SELECT COALESCE(MAX(score), -1) INTO mscore FROM submissions subs 
            WHERE subs.user_id = $1 AND subs.problem_id = $2;
    ELSE
        WITH subtask_max_scores AS (
            SELECT subtask_id, MAX(computed_score) AS best_score FROM submission_subtasks stks
                WHERE subtask_id IS NOT NULL 
                AND stks.user_id = $1 AND stks.problem_id = $2 
                GROUP BY subtask_id
        ) SELECT COALESCE(SUM(best_score), -1) INTO mscore FROM subtask_max_scores;
    END IF;
END;
$$ LANGUAGE plpgsql STABLE PARALLEL SAFE;

CREATE TABLE IF NOT EXISTS max_scores (
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    problem_id bigint NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    score decimal,

    CONSTRAINT unique_max_score_pair UNIQUE (user_id, problem_id)
);

CREATE INDEX IF NOT EXISTS user_max_scores ON max_scores(user_id);
-- used for problem statistics
CREATE INDEX IF NOT EXISTS problem_max_scores ON max_scores(problem_id);

DELETE FROM max_scores;
WITH pairs AS (
    SELECT DISTINCT user_id, problem_id FROM submissions
) INSERT INTO max_scores (user_id, problem_id, score) SELECT user_id, problem_id, max_score(user_id, problem_id) FROM pairs;

CREATE OR REPLACE FUNCTION reset_max_score(user_id bigint, problem_id bigint) RETURNS void AS $$
    INSERT INTO max_scores (user_id, problem_id, score) VALUES ($1, $2, max_score($1, $2)) ON CONFLICT ON CONSTRAINT unique_max_score_pair DO UPDATE SET score = max_score($1, $2);
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION score_update_handler() RETURNS TRIGGER AS $$
BEGIN
    RETURN(SELECT reset_max_score(NEW.user_id, NEW.problem_id));
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION score_delete_handler() RETURNS TRIGGER AS $$
BEGIN
    RETURN(SELECT reset_max_score(OLD.user_id, OLD.problem_id));
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER submission_score_update
    AFTER INSERT OR UPDATE OF score
    ON submissions
    FOR EACH ROW
    EXECUTE FUNCTION score_update_handler();

CREATE OR REPLACE TRIGGER submission_score_delete
    AFTER DELETE
    ON submissions
    FOR EACH ROW
    EXECUTE FUNCTION score_delete_handler();

COMMIT;

-- Sync checkup query:
-- (SELECT * FROM max_score_view EXCEPT SELECT * FROM max_scores) UNION ALL (SELECT * FROM max_scores EXCEPT SELECT * FROM max_score_view);
