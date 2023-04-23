
ALTER TABLE contest_registrations ADD COLUMN individual_start_at timestamptz;
ALTER TABLE contest_registrations ADD COLUMN individual_end_at timestamptz;

ALTER TABLE contests ADD COLUMN per_user_time integer NOT NULL DEFAULT 0; -- seconds
ALTER TABLE contests ADD COLUMN register_during_contest boolean NOT NULL DEFAULT false;


-- replace view with updated properties

CREATE OR REPLACE VIEW running_contests AS (
    SELECT * from contests WHERE contests.start_time <= NOW() AND NOW() <= contests.end_time
);

CREATE OR REPLACE VIEW problem_viewers AS
    (SELECT pbs.id as problem_id, 0 as user_id
        FROM problems pbs
        WHERE pbs.visible = true) -- Base case, problem is visible
    UNION ALL
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM problems pbs CROSS JOIN users 
        WHERE pbs.visible = true) -- Problem is visible
    UNION ALL
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM problems pbs CROSS JOIN users 
        WHERE users.admin = true) -- User is admin
    UNION ALL
    (SELECT problem_id, user_id FROM problem_user_access) -- Problem editors/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id) -- Contest testers/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, running_contests contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true 
        AND contests.per_user_time = 0 AND contests.register_during_contest = false) -- Visible, running, non-USACO contests with no registering during contest, for anons. TODO: Find alternative
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, running_contests contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true 
        AND contests.per_user_time = 0 AND contests.register_during_contest = false) -- Visible, running, non-USACO contests with no registering during contest
    UNION ALL
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()) -- Visible contests after they ended for anons. TODO: Find alternative
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()) -- Visible contests after they ended
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, running_contests contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id 
        AND contests.per_user_time = 0) -- Contest registrants during the contest for non-USACO style
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, running_contests contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id 
        AND contests.per_user_time > 0 AND users.individual_start_at IS NOT NULL); -- Contest registrants that started during the contest for USACO-style contests

