CREATE OR REPLACE VIEW max_score_view (user_id, problem_id, score)
    AS SELECT user_id, problem_id, MAX(score) AS score FROM submissions
    GROUP BY problem_id, user_id ORDER BY problem_id, user_id;