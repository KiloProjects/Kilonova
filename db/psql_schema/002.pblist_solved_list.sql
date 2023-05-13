

CREATE OR REPLACE VIEW pblist_user_solved (user_id, list_id, count) AS 
    SELECT msv.user_id, pbs.list_id, COUNT(pbs.list_id) 
    FROM problem_list_deep_problems pbs, max_score_view msv 
    WHERE msv.score = 100 AND msv.problem_id = pbs.problem_id 
    GROUP BY list_id, user_id;
