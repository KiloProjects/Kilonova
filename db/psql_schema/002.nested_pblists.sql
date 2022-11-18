
CREATE TABLE IF NOT EXISTS problem_list_pblists (
    parent_id bigint NOT NULL REFERENCES problem_lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    child_id bigint NOT NULL REFERENCES problem_lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
    position bigint NOT NULL DEFAULT 0,
    UNIQUE (parent_id, child_id),
	CHECK (parent_id != child_id)
);

