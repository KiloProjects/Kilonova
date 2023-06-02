
CREATE MATERIALIZED VIEW problem_statistics (problem_id, num_attempted, num_solved) AS
    SELECT problem_id, COUNT(*) AS num_attempted, COUNT(*) FILTER (WHERE score = 100) AS num_solved FROM max_score_view WHERE score != -1 GROUP BY problem_id;
