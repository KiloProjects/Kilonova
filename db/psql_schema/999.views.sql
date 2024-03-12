BEGIN;

-- NOTE: This file will never be inside 001.base.sql
-- This can be rerun every time a type or column is changed so the views will be refreshed

DROP VIEW IF EXISTS max_score_view CASCADE;
CREATE OR REPLACE VIEW max_score_view (user_id, problem_id, score) AS 
    WITH max_submission_strat AS (
        SELECT user_id, problem_id, MAX(score) AS max_score FROM submissions GROUP BY user_id, problem_id
    ), subtask_max_scores AS (
        SELECT DISTINCT user_id, subtask_id, problem_id, FIRST_VALUE(computed_score) OVER w AS max_score, FIRST_VALUE(created_at) OVER w AS mintime
        FROM submission_subtasks stks
        WHERE subtask_id IS NOT NULL 
            WINDOW w AS (PARTITION BY user_id, subtask_id, problem_id ORDER BY computed_score DESC, created_at ASC)
    ), sum_subtasks_strat AS (
        SELECT user_id, problem_id, coalesce(SUM(max_score), -1) AS max_score FROM subtask_max_scores GROUP BY user_id, problem_id
    ), user_subs AS (
        SELECT DISTINCT user_id, problem_id FROM submissions
    ) SELECT subs.user_id user_id, 
            pbs.id problem_id, 
            CASE WHEN pbs.scoring_strategy = 'max_submission' OR pbs.scoring_strategy = 'acm-icpc' THEN COALESCE(ms_sub.max_score, -1)
                 WHEN pbs.scoring_strategy = 'sum_subtasks'   THEN COALESCE(ms_subtask.max_score, -1)
                 ELSE -1
            END score
    FROM (user_subs subs JOIN problems pbs ON subs.problem_id = pbs.id) 
        LEFT JOIN max_submission_strat ms_sub ON (ms_sub.user_id = subs.user_id AND ms_sub.problem_id = pbs.id)
        LEFT JOIN sum_subtasks_strat ms_subtask ON (ms_subtask.user_id = subs.user_id AND ms_subtask.problem_id = pbs.id);

DROP VIEW IF EXISTS running_contests CASCADE;
CREATE OR REPLACE VIEW running_contests AS (
    SELECT * from contests WHERE contests.start_time <= NOW() AND NOW() <= contests.end_time
);

DROP MATERIALIZED VIEW IF EXISTS problem_statistics;
CREATE MATERIALIZED VIEW IF NOT EXISTS problem_statistics (problem_id, num_attempted, num_solved) AS
    SELECT problem_id, COUNT(*) AS num_attempted, COUNT(*) FILTER (WHERE score = 100) AS num_solved FROM max_scores WHERE score != -1 GROUP BY problem_id;

DROP FUNCTION IF EXISTS visible_pbs CASCADE;
-- param 1: the user ID for which we want to see the visible problems
-- guarantee: user_id in the returned table is equal to the user_id supplied
CREATE OR REPLACE FUNCTION visible_pbs(user_id bigint) RETURNS TABLE (problem_id bigint, user_id bigint) AS $$
    (SELECT pbs.id as problem_id, 0 as user_id
        FROM problems pbs
        WHERE pbs.visible = true AND 0 = $1) -- Base case, problem is visible
    UNION ALL
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM users CROSS JOIN problems pbs
        WHERE (pbs.visible = true OR users.admin = true) AND users.id = $1) -- Problem is visible or user is admin
    UNION ALL
    (SELECT problem_id, user_id FROM problem_user_access WHERE user_id = $1) -- Problem editors/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id AND users.user_id = $1) -- Contest testers/viewers TODO: maybe mark only for official?
    UNION ALL
    (SELECT pbs.problem_id as problem_id, 0 as user_id
        FROM contest_problems pbs, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true AND contests.type = 'official'
        AND contests.end_time <= NOW()
        AND 0 = $1) -- Visible OFFICIAL contests after they ended for anons. TODO: Find alternative
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.id as user_id
        FROM contest_problems pbs, users, contests
        WHERE pbs.contest_id = contests.id AND contests.visible = true AND contests.type = 'official'
        AND contests.end_time <= NOW()
        AND users.id = $1) -- Visible OFFICIAL contests after they ended
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
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, running_contests contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id 
        AND contests.per_user_time = 0
        AND users.user_id = $1) -- Contest registrants during the contest for non-USACO style
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id
        FROM contest_problems pbs, contest_registrations users, running_contests contests 
        WHERE pbs.contest_id = contests.id AND contests.id = users.contest_id 
        AND contests.per_user_time > 0 AND users.individual_start_at IS NOT NULL AND NOW() <= users.individual_end_at
        AND users.user_id = $1); -- Contest registrants that started during the contest for USACO-style contests
$$ LANGUAGE SQL STABLE;

DROP FUNCTION IF EXISTS visible_posts CASCADE;
-- Note: This must be in sync with sudoapi/filters.go:IsBlogPostVisible
CREATE OR REPLACE FUNCTION visible_posts(user_id bigint) RETURNS TABLE (post_id bigint, user_id bigint) AS $$
    SELECT id AS post_id, $1 AS user_id FROM blog_posts 
        WHERE 
            visible = true OR 
            EXISTS (SELECT 1 FROM users WHERE id = $1 AND admin = true) OR 
            author_id = $1
$$ LANGUAGE SQL STABLE;

DROP FUNCTION IF EXISTS persistently_visible_pbs CASCADE;
-- param 1: the user ID for which we want to see the persistently visible problems
-- persistently visible problems are problems that aren't conditioned on contest participation to be visible
CREATE OR REPLACE FUNCTION persistently_visible_pbs(user_id bigint) RETURNS TABLE (problem_id bigint, user_id bigint) AS $$
    (SELECT pbs.id as problem_id, 0 as user_id
        FROM problems pbs
        WHERE pbs.visible = true AND 0 = $1) -- Base case, problem is visible
    UNION ALL
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM users CROSS JOIN problems pbs
        WHERE (pbs.visible = true OR users.admin = true) AND users.id = $1) -- Problem is visible or user is admin
    UNION ALL
    (SELECT problem_id, user_id FROM problem_user_access WHERE user_id = $1) -- Problem editors/viewers
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users 
        WHERE pbs.contest_id = users.contest_id AND users.user_id = $1) -- Contest testers/viewers
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
        AND users.id = $1); -- Visible contests after they ended
$$ LANGUAGE SQL STABLE;

DROP VIEW IF EXISTS problem_editors CASCADE;
CREATE OR REPLACE VIEW problem_editors AS
    (SELECT pbs.id as problem_id, users.id as user_id 
        FROM problems pbs, users 
        WHERE users.admin = true) -- User is admin
    UNION ALL
    (SELECT problem_id, user_id FROM problem_user_access WHERE access = 'editor') -- Problem editors
    UNION ALL
    (SELECT pbs.problem_id as problem_id, users.user_id as user_id 
        FROM contest_problems pbs, contest_user_access users, contests
        WHERE contests.id = pbs.contest_id AND pbs.contest_id = users.contest_id 
            AND users.access = 'editor' AND contests.type = 'official'); -- Contest editors in official contests

-- Cases where a contest should be visible:
--   - It's visible
--   - Admins
--   - Testers/Editors
--   - It's not visible but it's running and user is registered
CREATE OR REPLACE FUNCTION visible_contests(user_id bigint) RETURNS TABLE (contest_id bigint, user_id bigint) AS $$
    (SELECT contests.id AS contest_id, $1 AS user_id FROM contests 
        WHERE contests.visible = true AND 0 = $1) -- visible to anonymous users
    UNION
    (SELECT contests.id AS contest_id, users.id AS user_id FROM contests, users 
        WHERE (contests.visible = true OR users.admin = true) AND users.id = $1) -- visible to logged in users and admins
    UNION
    (SELECT contest_id, user_id FROM contest_user_access WHERE user_id = $1) -- Testers/Editors
    UNION
    (SELECT contests.id AS contest_id, users.user_id AS user_id FROM contests, contest_registrations users 
        WHERE contests.id = users.contest_id AND contests.visible = false AND users.user_id = $1) -- not visible but registered
$$ LANGUAGE SQL STABLE;

DROP FUNCTION IF EXISTS contest_max_scores(bigint);
DROP FUNCTION IF EXISTS contest_max_scores(bigint, timestamptz);
CREATE OR REPLACE FUNCTION contest_max_scores(contest_id bigint, freeze_time timestamptz) RETURNS TABLE(user_id bigint, problem_id bigint, score decimal, mintime timestamptz) AS $$
    WITH max_submission_strat AS (
        SELECT DISTINCT user_id, problem_id, FIRST_VALUE(score * (leaderboard_score_scale / 100)) OVER w AS max_score, FIRST_VALUE(created_at) OVER w AS mintime
            FROM submissions WHERE contest_id = $1 AND created_at <= COALESCE(freeze_time, NOW()) AND (status = 'finished' OR status = 'reevaling')
            WINDOW w AS (PARTITION BY user_id, problem_id ORDER BY score DESC, created_at ASC)
    ), subtask_max_scores AS (
        SELECT DISTINCT user_id, subtask_id, problem_id, FIRST_VALUE(computed_score * (leaderboard_score_scale / 100)) OVER w AS max_score, FIRST_VALUE(created_at) OVER w AS mintime
        FROM submission_subtasks stks
        WHERE subtask_id IS NOT NULL AND contest_id = $1
            AND created_at <= COALESCE(freeze_time, NOW())
            WINDOW w AS (PARTITION BY user_id, subtask_id, problem_id ORDER BY computed_score DESC, created_at ASC)
    ), sum_subtasks_strat AS (
        SELECT DISTINCT user_id, problem_id, coalesce(SUM(max_score), -1) AS max_score, MAX(mintime) AS mintime FROM subtask_max_scores GROUP BY user_id, problem_id
    ) SELECT 
        users.user_id user_id,
        pbs.problem_id problem_id,
        CASE WHEN problems.scoring_strategy = 'max_submission' OR problems.scoring_strategy = 'acm-icpc' THEN COALESCE(ms_sub.max_score, -1)
            WHEN problems.scoring_strategy = 'sum_subtasks'   THEN COALESCE(ms_subtask.max_score, -1)
            ELSE -1
        END score,
        CASE WHEN problems.scoring_strategy = 'max_submission' OR problems.scoring_strategy = 'acm-icpc' THEN COALESCE(ms_sub.mintime, NULL)
            WHEN problems.scoring_strategy = 'sum_subtasks'   THEN COALESCE(ms_subtask.mintime, NULL)
            ELSE NULL
        END mintime
    FROM ((contest_problems pbs INNER JOIN contest_registrations users ON users.contest_id = pbs.contest_id AND pbs.contest_id = $1) INNER JOIN problems ON pbs.problem_id = problems.id)
        LEFT JOIN max_submission_strat ms_sub ON (ms_sub.user_id = users.user_id AND ms_sub.problem_id = pbs.problem_id)
        LEFT JOIN sum_subtasks_strat ms_subtask ON (ms_subtask.user_id = users.user_id AND ms_subtask.problem_id = pbs.problem_id)
$$ LANGUAGE SQL STABLE;

DROP VIEW IF EXISTS contest_submission_subtask_max_scores CASCADE;
DROP VIEW IF EXISTS max_score_contest_view CASCADE;

DROP VIEW IF EXISTS contest_top_view CASCADE;
DROP FUNCTION IF EXISTS contest_top_view;
-- Since we now return -1 on no attempt, we must filter it when computing the top view
-- also, exclude contest editors/testers since they didn't get that score legit
CREATE OR REPLACE FUNCTION contest_top_view(contest_id bigint, freeze_time timestamptz, include_editors boolean) RETURNS TABLE (user_id bigint, contest_id bigint, total_score decimal, last_time timestamptz) AS $$
    -- both contest_scores and legit_contestants will contain results only for that contest id, so it's safe to simply join them 
    WITH contest_scores AS (
        SELECT user_id, SUM(score) AS total_score, MAX(mintime) FILTER (WHERE score > 0) AS last_time FROM contest_max_scores($1, $2) WHERE score >= 0 GROUP BY user_id
    ), legit_contestants AS (
        SELECT regs.* FROM contest_registrations regs WHERE regs.contest_id = $1 AND (NOT EXISTS (SELECT 1 FROM contest_user_access acc WHERE acc.user_id = regs.user_id AND acc.contest_id = regs.contest_id) OR $3 = true)
    )
    SELECT users.user_id, $1 AS contest_id, COALESCE(scores.total_score, 0) AS total_score, last_time
    FROM 
        legit_contestants users 
        LEFT JOIN contest_scores scores ON users.user_id = scores.user_id 
        ORDER BY COALESCE(total_score, 0) DESC, last_time ASC NULLS LAST, user_id;
$$ LANGUAGE SQL STABLE;

DROP FUNCTION IF EXISTS contest_icpc_view;
-- we exclude contest editors/testers since they didn't get that score legit
CREATE OR REPLACE FUNCTION contest_icpc_view(contest_id bigint, freeze_time timestamptz, include_editors boolean) 
RETURNS TABLE (user_id bigint, contest_id bigint, last_time timestamptz, num_solved integer, penalty integer, num_attempts integer) AS $$
    WITH legit_contestants AS (
        SELECT regs.* FROM contest_registrations regs WHERE regs.contest_id = $1 AND (NOT EXISTS (SELECT 1 FROM contest_user_access acc WHERE acc.user_id = regs.user_id AND acc.contest_id = regs.contest_id) OR $3 = true)
    ), solved_pbs AS (
        SELECT user_id, problem_id, mintime AS last_time FROM contest_max_scores($1, $2) WHERE score = 100
    ), last_times AS (
        SELECT user_id, MAX(last_time) AS last_time FROM solved_pbs GROUP BY user_id
    ), num_solved AS (
        SELECT user_id, COUNT(*) AS num_problems, MAX(last_time) AS mintime FROM solved_pbs GROUP BY user_id
    ), num_attempts AS (
        -- TODO: Keep kind of in sync with left join in icpcToLeaderboardEntry
        SELECT solved_pbs.user_id, COUNT(*) AS num_attempts 
            FROM solved_pbs 
            INNER JOIN submissions subs ON subs.contest_id = $1 
                AND subs.user_id = solved_pbs.user_id 
                AND subs.problem_id = solved_pbs.problem_id 
                AND subs.created_at < solved_pbs.last_time
                AND (subs.status = 'finished' OR subs.status = 'reevaling')
            GROUP BY solved_pbs.user_id
    ), duration_sum AS (
        -- get sum of number of minutes from problems
        SELECT user_id, 
            SUM(FLOOR(EXTRACT(EPOCH FROM last_time - (SELECT start_time FROM contests WHERE id = $1 LIMIT 1)) / 60)) AS mins 
        FROM solved_pbs GROUP BY user_id
    ), penalties AS (
        SELECT users.user_id, 
            COALESCE(atts.num_attempts, 0) * (SELECT icpc_submission_penalty FROM contests WHERE id = $1 LIMIT 1) 
                + GREATEST(COALESCE(dsum.mins, 0), 0)
                AS penalty, atts.num_attempts AS num_attempts
        FROM legit_contestants users
            LEFT JOIN num_attempts atts ON atts.user_id = users.user_id
            LEFT JOIN duration_sum dsum ON dsum.user_id = users.user_id
    ) SELECT users.user_id, $1 AS contest_id, last_time, COALESCE(num_problems, 0), COALESCE(penalty, 0), COALESCE(num_attempts, 0)
        FROM legit_contestants users
        LEFT JOIN last_times ON users.user_id = last_times.user_id
        LEFT JOIN num_solved ON users.user_id = num_solved.user_id
        LEFT JOIN penalties ON users.user_id = penalties.user_id
    ORDER BY COALESCE(num_problems, 0) DESC, penalty ASC NULLS LAST, last_time ASC NULLS LAST, user_id;
$$ LANGUAGE SQL STABLE;

DROP FUNCTION IF EXISTS visible_submissions;
CREATE OR REPLACE FUNCTION visible_submissions(user_id bigint) RETURNS TABLE (sub_id bigint) AS $$
    WITH v_pbs AS (SELECT * FROM visible_pbs($1))
    (SELECT id as sub_id
        FROM submissions, v_pbs
        WHERE submissions.user_id = $1 AND submissions.problem_id = v_pbs.problem_id) -- base case, users should see their own submissions if problem is still visible (also, coincidentally, works for contest problems)
    UNION ALL
    (SELECT subs.id as sub_id
        FROM submissions subs, v_pbs pb_viewers
        WHERE pb_viewers.problem_id = subs.problem_id AND subs.contest_id IS NULL) -- contest is null, so judge if problem is visible
    UNION ALL
    (SELECT subs.id as sub_id
        FROM submissions subs, contest_user_access users
        WHERE users.contest_id = subs.contest_id AND users.user_id = $1 AND subs.contest_id IS NOT NULL) -- contest staff if contest is not null
    UNION ALL
    (SELECT subs.id as sub_id
        FROM submissions subs, contests, v_pbs
        WHERE EXISTS (SELECT 1 FROM visible_contests($1) viz WHERE contests.id = viz.contest_id)
        AND contests.id = subs.contest_id AND v_pbs.problem_id = subs.problem_id
        AND contests.end_time <= NOW()) -- if the contest ended and the problem is visible, show the submission
$$ LANGUAGE SQL STABLE;

DROP FUNCTION IF EXISTS visible_submissions_ex;
CREATE OR REPLACE FUNCTION visible_submissions_ex(user_id bigint, problem_id bigint, author_id bigint) RETURNS TABLE (sub_id bigint) AS $$
    WITH v_pbs AS (SELECT * FROM visible_pbs($1) WHERE $2 IS NULL OR problem_id = $2)
    (SELECT subs.id as sub_id
        FROM submissions subs, v_pbs
        WHERE subs.user_id = $1 AND subs.problem_id = v_pbs.problem_id AND ($3 IS NULL OR subs.user_id = $3) ) -- base case, users should see their own submissions if problem is still visible (also, coincidentally, works for contest problems)
    UNION ALL
    (SELECT subs.id as sub_id
        FROM submissions subs, v_pbs pb_viewers
        WHERE pb_viewers.problem_id = subs.problem_id AND subs.contest_id IS NULL AND ($3 IS NULL OR subs.user_id = $3)) -- contest is null, so judge if problem is visible
    UNION ALL
    (SELECT subs.id as sub_id
        FROM submissions subs, contest_user_access users
        WHERE users.contest_id = subs.contest_id AND users.user_id = $1 AND subs.contest_id IS NOT NULL) -- contest staff if contest is not null
    UNION ALL
    (SELECT subs.id as sub_id
        FROM submissions subs, contests, v_pbs
        WHERE EXISTS (SELECT 1 FROM visible_contests($1) viz WHERE contests.id = viz.contest_id)
        AND contests.id = subs.contest_id AND v_pbs.problem_id = subs.problem_id AND ($3 IS NULL OR subs.user_id = $3)
        AND contests.end_time <= NOW()) -- if the contest ended and the problem is visible, show the submission
$$ LANGUAGE SQL STABLE;

DROP VIEW IF EXISTS problem_list_deep_problems CASCADE;
CREATE MATERIALIZED VIEW IF NOT EXISTS problem_list_deep_problems (list_id, problem_id) AS
    WITH RECURSIVE pblist_tree(list_id, problem_id) AS (
        SELECT pblist_id AS list_id, problem_id FROM problem_list_problems
        UNION
        SELECT pbs.parent_id AS list_id, pt.problem_id FROM problem_list_pblists pbs, pblist_tree pt WHERE pbs.child_id = pt.list_id
    ) SELECT * FROM pblist_tree;

DROP VIEW IF EXISTS problem_list_deep_sublists;
CREATE OR REPLACE VIEW problem_list_deep_sublists (parent_id, child_id) AS 
    WITH RECURSIVE pblist_tree (parent_id, child_id) AS (
        SELECT id AS parent_id, id AS child_id FROM problem_lists
        UNION
        SELECT pbs.parent_id AS parent_id, pt.child_id FROM problem_list_pblists pbs, pblist_tree pt WHERE pbs.child_id = pt.parent_id
    ) SELECT * FROM pblist_tree;

DROP MATERIALIZED VIEW IF EXISTS problem_list_pb_count;
CREATE MATERIALIZED VIEW IF NOT EXISTS problem_list_pb_count(list_id, count) AS 
    WITH RECURSIVE pblist_tree(list_id, problem_id) AS (
        SELECT pblist_id AS list_id, problem_id FROM problem_list_problems
        UNION
        SELECT pbs.parent_id AS list_id, pt.problem_id FROM problem_list_pblists pbs, pblist_tree pt WHERE pbs.child_id = pt.list_id
    ) SELECT list_id, COUNT(*) FROM pblist_tree GROUP BY list_id;

CREATE OR REPLACE FUNCTION refresh_pblist_pb_count() RETURNS TRIGGER AS $$
    BEGIN
        REFRESH MATERIALIZED VIEW problem_list_pb_count;
        REFRESH MATERIALIZED VIEW problem_list_deep_problems;
        RETURN NULL;
    END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER pblist_problem_count_on_pbs
    AFTER INSERT OR UPDATE OR DELETE
    ON problem_list_problems
    FOR EACH STATEMENT
    EXECUTE FUNCTION refresh_pblist_pb_count();
CREATE OR REPLACE TRIGGER pblist_problem_count_on_pblists
    AFTER INSERT OR UPDATE OR DELETE
    ON problem_list_pblists
    FOR EACH STATEMENT
    EXECUTE FUNCTION refresh_pblist_pb_count();

DROP VIEW IF EXISTS pblist_user_solved;
CREATE OR REPLACE VIEW pblist_user_solved (user_id, list_id, count) AS 
    SELECT msv.user_id, pbs.list_id, COUNT(pbs.list_id) 
    FROM problem_list_deep_problems pbs, max_scores msv 
    WHERE msv.score = 100 AND msv.problem_id = pbs.problem_id 
    GROUP BY list_id, user_id;


DROP VIEW IF EXISTS problem_checklist;
-- note that, because max_scores only counts *users* that completed the problem, the num_sols is actually the number of users that completed the problem
-- NOT the number of 100-point solutions (but this metric would be irrelevant in the max-subtasks scoring strategy)
CREATE OR REPLACE VIEW problem_checklist (problem_id, num_pdf, num_md, num_tests, num_subtasks, has_source, num_authors, num_other_tags, num_sols) AS 
    WITH problem_attachments AS (
        SELECT  atts.name AS name,
                pam.problem_id AS problem_id 
        FROM attachments atts, problem_attachments_m2m pam 
        WHERE atts.id = pam.attachment_id
    )
    SELECT 
        pbs.id AS problem_id,
        (SELECT COUNT(*) FROM problem_attachments WHERE name LIKE 'statement-%.pdf' AND problem_id = pbs.id) AS num_pdf,
        (SELECT COUNT(*) FROM problem_attachments WHERE name LIKE 'statement-%.md' AND problem_id = pbs.id) AS num_md,
        (SELECT COUNT(*) FROM tests WHERE problem_id = pbs.id) AS num_tests,
        (SELECT COUNT(*) FROM subtasks WHERE problem_id = pbs.id) AS num_subtasks,
        (length(btrim(pbs.source_credits)) > 0) has_source,
        (SELECT COUNT(*) FROM problem_tags ptags WHERE EXISTS (SELECT 1 FROM tags WHERE tags.id = ptags.tag_id AND tags.type = 'author') AND problem_id = pbs.id) AS num_authors,
        (SELECT COUNT(*) FROM problem_tags ptags WHERE EXISTS (SELECT 1 FROM tags WHERE tags.id = ptags.tag_id AND tags.type != 'author') AND problem_id = pbs.id) AS num_other_tags,
        (SELECT COUNT(*) FROM max_scores WHERE problem_id = pbs.id AND score = 100) AS num_sols
    FROM problems pbs;

COMMIT;