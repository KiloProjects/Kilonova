
-- param 1: the user ID for which we want to see the visible problems
CREATE OR REPLACE FUNCTION visible_pbs(user_id bigint) RETURNS TABLE (problem_id bigint, user_id bigint) AS $$
    (SELECT pbs.id as problem_id, 0 as user_id
        FROM problems pbs
        WHERE pbs.visible = true AND 0 = $1) -- Base case, problem is visible
    UNION ALL
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM users CROSS JOIN problems pbs
        WHERE pbs.visible = true AND users.id = $1) -- Problem is visible
    UNION ALL
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM users CROSS JOIN problems pbs 
        WHERE users.admin = true AND users.id = $1) -- User is admin
    UNION ALL
    (SELECT problem_id, user_id FROM problem_user_access WHERE user_id = $1) -- Problem editors/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id AND users.user_id = $1) -- Contest testers/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, running_contests contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true 
        AND contests.per_user_time = 0 AND contests.register_during_contest = false
        AND 0 = $1) -- Visible, running, non-USACO contests with no registering during contest, for anons. TODO: Find alternative
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, running_contests contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true 
        AND contests.per_user_time = 0 AND contests.register_during_contest = false
        AND users.id = $1) -- Visible, running, non-USACO contests with no registering during contest
    UNION ALL
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()
        AND 0 = $1) -- Visible contests after they ended for anons. TODO: Find alternative
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true
        AND contests.end_time <= NOW()
        AND users.id = $1) -- Visible contests after they ended
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, running_contests contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id 
        AND contests.per_user_time = 0
        AND users.user_id = $1) -- Contest registrants during the contest for non-USACO style
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, running_contests contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id 
        AND contests.per_user_time > 0 AND users.individual_start_at IS NOT NULL
        AND users.user_id = $1); -- Contest registrants that started during the contest for USACO-style contests
$$ LANGUAGE SQL STABLE;



-- make submission_viewers make use of this by rewriting it in a function as well

CREATE OR REPLACE FUNCTION visible_submissions(user_id bigint) RETURNS TABLE (sub_id bigint, user_id bigint) AS $$
    (SELECT id as sub_id, user_id as user_id 
        FROM submissions 
        WHERE user_id = $1 AND EXISTS (
            SELECT 1 
            FROM visible_pbs($1) 
            WHERE problem_id = submissions.problem_id AND user_id = $1
        )) -- base case, users should see their own submissions if problem is still visible (also, coincidentally, works for contest problems)
    UNION ALL
    (SELECT subs.id as sub_id, users.id as user_id
        FROM submissions subs, users
        WHERE users.admin = true AND users.id = $1) -- user admins
    UNION ALL
    (SELECT subs.id as sub_id, pb_viewers.user_id as user_id
        FROM submissions subs, visible_pbs($1) pb_viewers
        WHERE pb_viewers.problem_id = subs.problem_id AND subs.contest_id IS null
        AND pb_viewers.user_id = $1) -- contest is null, so judge if problem is visible
    UNION ALL
    (SELECT subs.id as sub_id, users.user_id as user_id
        FROM submissions subs, contest_user_access users
        WHERE users.contest_id = subs.contest_id AND users.access = 'editor'
        AND users.user_id = $1) -- contest editors if contest is not null
    UNION ALL
    (SELECT subs.id as sub_id, viz.user_id as user_id
        FROM submissions subs, contests, contest_visibility viz, contest_problems c_pbs
        WHERE contests.id = subs.contest_id AND contests.id = viz.contest_id AND contests.id = c_pbs.contest_id
        AND c_pbs.problem_id = subs.problem_id
        AND contests.end_time <= NOW()
        AND viz.user_id = $1) -- ended contest, so everyone that can see the contest can also see submission
            -- (if they can see the contest, they can also see the problem, since it ended)
$$ LANGUAGE SQL STABLE;

DROP VIEW submission_viewers;
DROP VIEW problem_viewers;
