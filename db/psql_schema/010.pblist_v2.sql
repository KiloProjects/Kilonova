BEGIN;

CREATE TABLE problem_list_problems (
    pblist_id bigint NOT NULL REFERENCES problem_lists(id) ON DELETE CASCADE ON UPDATE CASCADE, 
    problem_id bigint NOT NULL REFERENCES problems(id) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE (pblist_id, problem_id)
);

INSERT INTO problem_list_problems SELECT pl1.id AS pblist_id, pp.ns AS problem_id FROM problem_lists pl1, LATERAL (SELECT unnest(list) AS ns FROM problem_lists pl2 WHERE pl1.id = pl2.id) pp;

CREATE TABLE subtask_tests (
    subtask_id bigint NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    test_id bigint NOT NULL REFERENCES tests(id) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE (subtask_id, test_id)
);

INSERT INTO subtask_tests SELECT st1.id AS subtask_id, ss.ns AS test_id FROM subtasks st1, LATERAL (SELECT unnest(tests) AS ns FROM subtasks st2 WHERE st1.id = st2.id) ss;

ALTER TABLE problem_lists DROP COLUMN list;
ALTER TABLE subtasks DROP COLUMN tests;

COMMIT;
