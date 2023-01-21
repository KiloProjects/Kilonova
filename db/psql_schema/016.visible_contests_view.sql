-- Cases where a contest should be visible:
--   - It's visible
--   - Admins
--   - Testers/Editors
--   - It's not visible but it's running and user is registered
CREATE VIEW contest_visibility AS (
    (SELECT contests.id AS contest_id, 0 AS user_id FROM contests 
        WHERE contests.visible = true) -- visible
    UNION
    (SELECT contests.id AS contest_id, users.id AS user_id FROM contests, users 
        WHERE users.admin = true) -- admin
    UNION
    (SELECT contest_id, user_id FROM contest_user_access) -- Testers/Editors
    UNION
    (SELECT contests.id AS contest_id, users.user_id AS user_id FROM contests, contest_registrations users 
        WHERE contests.id = users.contest_id AND contests.visible = false
            AND contests.start_time <= NOW() AND NOW() <= contests.end_time) -- not visible but registered and running
);