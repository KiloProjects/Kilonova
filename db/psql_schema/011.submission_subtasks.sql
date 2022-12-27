
BEGIN;

CREATE TABLE IF NOT EXISTS submission_subtasks (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT NOW(),
    submission_id   bigint          NOT NULL REFERENCES submissions(id) ON DELETE CASCADE ON UPDATE CASCADE,
    
    -- user_id might be useful in the future
    -- if I ever implement a cross-submission total score system
    user_id         bigint          NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    
    -- Might be useful to also store subtask id, just in case, even if it may never be properly used
    subtask_id      bigint          REFERENCES subtasks(id) ON DELETE SET NULL ON UPDATE CASCADE,

    -- Copy attributes from subtask
	problem_id 	    bigint 		    NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	visible_id 	    integer 	    NOT NULL,
	score 		    integer 	    NOT NULL,
    UNIQUE (submission_id, subtask_id)
);

CREATE TABLE IF NOT EXISTS submission_subtask_subtests (
    submission_subtask_id   bigint NOT NULL REFERENCES submission_subtasks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    submission_test_id      bigint NOT NULL REFERENCES submission_tests(id) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE (submission_subtask_id, submission_test_id)
);

-- Add more attributes from subtest to test
ALTER TABLE submission_tests ADD COLUMN visible_id bigint NULL;
ALTER TABLE submission_tests ADD COLUMN max_score  bigint NULL;
WITH cte AS (
    SELECT id, score, visible_id FROM tests
) UPDATE submission_tests 
SET visible_id = cte.visible_id, max_score = cte.score
FROM cte WHERE cte.id = submission_tests.test_id;
ALTER TABLE submission_tests ALTER COLUMN visible_id SET NOT NULL;
ALTER TABLE submission_tests ALTER COLUMN max_score  SET NOT NULL;

-- Insert subtasks for submissions created before this change
WITH cte AS (
    SELECT subs.id AS submission_id, subs.user_id, stks.id AS subtask_id, subs.problem_id, stks.visible_id, stks.score 
    FROM submissions subs INNER JOIN subtasks stks ON stks.problem_id = subs.problem_id
)
INSERT INTO submission_subtasks (
    submission_id,
    user_id,
    subtask_id,
    problem_id,
    visible_id,
    score
) SELECT cte.submission_id, cte.user_id, cte.subtask_id, cte.problem_id, cte.visible_id, cte.score FROM cte;


-- Generate associations for submission subtasks created before this change
WITH cte AS (
    SELECT stks.id AS sub_stk_id, ststs.id AS subtest_id 
    FROM submission_subtasks stks 
        INNER JOIN subtask_tests ON stks.subtask_id = subtask_tests.subtask_id 
        INNER JOIN tests ON tests.id = subtask_tests.test_id 
        INNER JOIN submission_tests ststs 
            ON (
                (tests.id = ststs.test_id AND subtask_tests.test_id = ststs.test_id) 
                OR ststs.visible_id = tests.visible_id
            ) AND ststs.submission_id = stks.submission_id
)
INSERT INTO submission_subtask_subtests (
    submission_subtask_id,
    submission_test_id
) SELECT cte.sub_stk_id, cte.subtest_id FROM cte;


COMMIT;

-- TODO: Now that we have all the necessary attributes for a subtest copied into the subtest
-- Can't we update test_id to be nullable with on delete set default and remove all the orphaned cruft?
