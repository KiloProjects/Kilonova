
-- TODO: also have submission time??
CREATE OR REPLACE VIEW max_score_view (user_id, problem_id, score) AS 
    SELECT users.id user_id, pbs.id problem_id, ms.max_score score FROM (problems pbs CROSS JOIN users)
    LEFT JOIN LATERAL (SELECT coalesce(MAX(score), -1) AS max_score FROM submissions WHERE user_id = users.id AND problem_id = pbs.id) ms ON TRUE;
    