CREATE OR REPLACE VIEW problem_list_deep_problems (list_id, problem_id) AS
    WITH RECURSIVE pblist_tree(list_id, problem_id) AS (
        SELECT pblist_id AS list_id, problem_id FROM problem_list_problems
        UNION
        SELECT pbs.parent_id AS list_id, pt.problem_id FROM problem_list_pblists pbs, pblist_tree PT WHERE pbs.child_id = pt.list_id
    ) SELECT * FROM pblist_tree;

CREATE OR REPLACE VIEW problem_list_pb_count(list_id, count) AS 
    WITH RECURSIVE pblist_tree(list_id, problem_id) AS (
        SELECT pblist_id AS list_id, problem_id FROM problem_list_problems
        UNION
        SELECT pbs.parent_id AS list_id, pt.problem_id FROM problem_list_pblists pbs, pblist_tree PT WHERE pbs.child_id = pt.list_id
    ) SELECT list_id, COUNT(*) FROM pblist_tree GROUP BY list_id;

