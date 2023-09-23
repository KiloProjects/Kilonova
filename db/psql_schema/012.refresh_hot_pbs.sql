CREATE OR REPLACE FUNCTION refresh_hot_pbs(banned_problems bigint[]) RETURNS void AS $$
BEGIN
    DROP MATERIALIZED VIEW IF EXISTS hot_problems;
    EXECUTE format('CREATE MATERIALIZED VIEW hot_problems (problem_id, hot_cnt) AS 
		SELECT problem_id, COUNT(DISTINCT user_id) AS user_cnt FROM submissions
			WHERE created_at >= NOW() - ''7 days''::interval 
				AND problem_id != ANY(%L::bigint[])
			GROUP BY problem_id
		ORDER BY user_cnt DESC;', $1);
END;
$$ LANGUAGE plpgsql;

