BEGIN;

-- Copy orphaned that are not used in subtests (basically dead)
\copy (SELECT id, problem_id, visible_id FROM tests WHERE orphaned = true AND NOT EXISTS (SELECT 1 FROM submission_tests WHERE test_id = tests.id) ORDER BY id ASC) TO ~/orphaned_unused.csv;

-- Copy orphaned that are still in use in some subtests
\copy (SELECT id, problem_id, visible_id FROM tests WHERE orphaned = true AND EXISTS (SELECT 1 FROM submission_tests WHERE test_id = tests.id) ORDER BY id ASC) TO ~/orphaned_used.csv;

-- The above are used for removing the test files later

-- REMOVE ORPHANED TESTS!!!!
DELETE FROM tests WHERE orphaned = true;

-- Update problem checklist so it doesn't depend on orphaned tests anymore
CREATE OR REPLACE VIEW problem_checklist (problem_id, num_pdf, num_md, num_tests, num_subtasks, has_source, num_authors, num_other_tags, num_sols) AS 
    WITH pdf_cnt AS (
        SELECT problem_id, COUNT(*) AS cnt FROM problem_attachments WHERE name LIKE 'statement-%.pdf' GROUP BY problem_id
    ), md_cnt AS (
        SELECT problem_id, COUNT(*) AS cnt FROM problem_attachments WHERE name LIKE 'statement-%.md' GROUP BY problem_id
    ), test_cnt AS (
        SELECT problem_id, COUNT(*) AS cnt FROM tests GROUP BY problem_id
    ), stk_cnt AS (
        SELECT problem_id, COUNT(*) AS cnt FROM subtasks GROUP BY problem_id
    ), author_cnt AS (
        SELECT problem_id, COUNT(*) AS cnt FROM problem_tags 
            WHERE EXISTS (SELECT 1 FROM tags WHERE tags.id = problem_tags.tag_id AND tags.type = 'author') 
        GROUP BY problem_id
    ), tag_cnt AS (
        SELECT problem_id, COUNT(*) AS cnt FROM problem_tags 
            WHERE EXISTS (SELECT 1 FROM tags WHERE tags.id = problem_tags.tag_id AND tags.type != 'author') 
        GROUP BY problem_id
    ), success_cnt AS (
        SELECT problem_id, COUNT(*) AS cnt FROM max_score_view WHERE score = 100 GROUP BY problem_id
    )

    SELECT 
        pbs.id AS problem_id,
        COALESCE(pdf_cnt.cnt, 0) num_pdf,
        COALESCE(md_cnt.cnt, 0) num_md,
        COALESCE(test_cnt.cnt, 0) num_tests,
        COALESCE(stk_cnt.cnt, 0) num_subtasks,
        (length(btrim(pbs.source_credits)) > 0) has_source,
        COALESCE(author_cnt.cnt, 0) num_authors,
        COALESCE(tag_cnt.cnt, 0) num_other_tags,
        COALESCE(success_cnt.cnt, 0) num_sols
    FROM problems pbs 
        LEFT JOIN pdf_cnt ON pbs.id = pdf_cnt.problem_id
        LEFT JOIN md_cnt ON pbs.id = md_cnt.problem_id
        LEFT JOIN test_cnt ON pbs.id = test_cnt.problem_id
        LEFT JOIN stk_cnt ON pbs.id = stk_cnt.problem_id
        LEFT JOIN author_cnt ON pbs.id = author_cnt.problem_id
        LEFT JOIN tag_cnt ON pbs.id = tag_cnt.problem_id
        LEFT JOIN success_cnt ON pbs.id = success_cnt.problem_id;

ALTER TABLE tests DROP COLUMN orphaned;

COMMIT;
