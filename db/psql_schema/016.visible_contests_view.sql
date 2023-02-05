-- Cases where a contest should be visible:
--   - It's visible
--   - Admins
--   - Testers/Editors
--   - It's not visible but it's running and user is registered

CREATE OR REPLACE VIEW running_contests AS (
    SELECT * from contests WHERE contests.start_time <= NOW() AND NOW() <= contests.end_time
);

CREATE OR REPLACE VIEW contest_visibility AS (
    (SELECT contests.id AS contest_id, 0 AS user_id FROM contests 
        WHERE contests.visible = true) -- visible to anonymous users
    UNION
    (SELECT contests.id AS contest_id, users.id AS user_id FROM contests, users 
        WHERE contests.visible = true) -- visible to logged in users
    UNION
    (SELECT contests.id AS contest_id, users.id AS user_id FROM contests, users 
        WHERE users.admin = true) -- admin
    UNION
    (SELECT contest_id, user_id FROM contest_user_access) -- Testers/Editors
    UNION
    (SELECT contests.id AS contest_id, users.user_id AS user_id FROM contests, contest_registrations users 
        WHERE contests.id = users.contest_id AND contests.visible = false) -- not visible but registered
);