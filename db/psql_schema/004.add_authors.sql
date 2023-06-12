

BEGIN;

CREATE TEMPORARY VIEW problem_authors AS
    SELECT id, trim(unnest(string_to_array(author_credits::text, ','))) AS name FROM problems;

INSERT INTO tags (name, type) SELECT DISTINCT name, 'author'::tag_type FROM problem_authors;

INSERT INTO problem_tags (tag_id, problem_id) SELECT tags.id, problem_authors.id FROM tags, problem_authors WHERE tags.name = problem_authors.name;

ALTER TABLE problems DROP COLUMN author_credits CASCADE;

COMMIT;