CREATE OR REPLACE VIEW problem_viewers AS
    (SELECT pbs.id as problem_id, 0 as user_id
        FROM problems pbs, users
        WHERE pbs.visible = true) -- Base case, problem is visible
    UNION
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM problems pbs, users 
        WHERE pbs.visible = true OR users.admin = true) -- Problem is visible or user is admin
    UNION
    (SELECT problem_id, user_id FROM problem_user_access) -- Problem editors/viewers
    UNION
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id) -- Contest editors/viewers
    UNION
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.start_time <= NOW() AND NOW() <= contests.end_time) -- Visible running contests for anons. TODO: Find alternative
    UNION
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.start_time <= NOW() AND NOW() <= contests.end_time) -- Visible running contests
    UNION
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()) -- Visible contests after they ended for anons. TODO: Find alternative
    UNION
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()) -- Visible contests after they ended
    UNION
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id
        AND contests.visible = false 
        AND contests.start_time <= NOW() AND NOW() <= contests.end_time); -- Contest registrants during the contest for hidden running contest

CREATE OR REPLACE VIEW problem_editors AS
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM problems pbs, users 
        WHERE users.admin = true) -- User is admin
    UNION
    (SELECT problem_id, user_id FROM problem_user_access WHERE access = 'editor') -- Problem editors
    UNION
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id AND users.access = 'editor'); -- Contest editors
