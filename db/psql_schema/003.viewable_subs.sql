

CREATE OR REPLACE VIEW submission_viewers AS (
    (SELECT id as sub_id, user_id as user_id 
        FROM submissions 
        WHERE EXISTS (
            SELECT 1 
            FROM problem_viewers 
            WHERE problem_id = submissions.problem_id AND user_id = submissions.user_id
        )) -- base case, users should see their own submissions if problem is still visible (also, coincidentally, works for contest problems)
    UNION ALL
    (SELECT subs.id as sub_id, users.id as user_id
        FROM submissions subs, users
        WHERE users.admin = true) -- user admins
    UNION ALL
    (SELECT subs.id as sub_id, pb_viewers.user_id as user_id
        FROM submissions subs, problem_viewers pb_viewers
        WHERE pb_viewers.problem_id = subs.problem_id AND subs.contest_id IS null) -- contest is null, so judge if problem is visible
    UNION ALL
    (SELECT subs.id as sub_id, users.user_id as user_id
        FROM submissions subs, contest_user_access users
        WHERE users.contest_id = subs.contest_id AND users.access = 'editor') -- contest editors if contest is not null
    UNION ALL
    (SELECT subs.id as sub_id, viz.user_id as user_id
        FROM submissions subs, contests, contest_visibility viz, contest_problems c_pbs
        WHERE contests.id = subs.contest_id AND contests.id = viz.contest_id AND contests.id = c_pbs.contest_id
        AND c_pbs.problem_id = subs.problem_id
        AND contests.end_time <= NOW()) -- ended contest, so everyone that can see the contest can also see submission
            -- (if they can see the contest, they can also see the problem, since it ended)
);


CREATE INDEX IF NOT EXISTS problem_visibility_index ON problems (visible);
