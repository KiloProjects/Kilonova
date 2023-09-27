
-- NOTE: When merging sql files, do not copy this in 002.views.sql
-- The table is now generated automatically by refresh_hot_pbs function defined later
CREATE MATERIALIZED VIEW IF NOT EXISTS hot_problems (problem_id, hot_cnt) AS 
    SELECT problem_id, COUNT(DISTINCT user_id) AS user_cnt FROM submissions
        WHERE created_at >= NOW() - '7 days'::interval
        GROUP BY problem_id
        ORDER BY user_cnt DESC;
