

CREATE TYPE pbaccess_type AS ENUM (
    'editor',
    'viewer'
); 

CREATE TABLE IF NOT EXISTS problem_user_access (
    problem_id  bigint           NOT NULL REFERENCES problems(id) ON DELETE CASCADE ON UPDATE CASCADE,
    user_id     bigint           NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    access      pbaccess_type    NOT NULL,

    UNIQUE (problem_id, user_id)
);

INSERT INTO problem_user_access (problem_id, user_id, access) SELECT id as problem_id, author_id as user_id, 'editor' as access FROM problems;

ALTER TABLE problems DROP COLUMN author_id;
