
CREATE OR REPLACE VIEW max_score_contest_view
    AS SELECT subs.user_id, subs.problem_id, subs.contest_id, MAX(subs.score) AS score FROM submissions subs, contest_registrations users, contest_problems pbs 
    WHERE subs.contest_id IS NOT NULL -- actually from contest
    AND users.user_id = subs.user_id AND users.contest_id = subs.contest_id -- Registered users
    AND pbs.problem_id = subs.problem_id AND pbs.contest_id = subs.contest_id -- Existent problems
    GROUP BY subs.problem_id, subs.contest_id, subs.user_id ORDER BY subs.contest_id, subs.user_id, subs.problem_id;

CREATE OR REPLACE VIEW contest_top_view
    AS SELECT user_id, contest_id, SUM(score) AS total_score FROM max_score_contest_view GROUP BY user_id, contest_id ORDER BY contest_id, total_score DESC, user_id; 