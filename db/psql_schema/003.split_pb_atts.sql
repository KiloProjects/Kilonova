
BEGIN;

ALTER TABLE attachments ADD COLUMN last_updated_at timestamptz NOT NULL DEFAULT NOW();
ALTER TABLE attachments ADD COLUMN last_updated_by bigint REFERENCES users(id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS problem_attachments_m2m (
	problem_id 	    bigint 		NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
	attachment_id 	bigint 		NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
    UNIQUE (problem_id, attachment_id)
);

INSERT INTO problem_attachments_m2m (problem_id, attachment_id) SELECT problem_id, id FROM attachments;

ALTER TABLE attachments DROP COLUMN problem_id;

CREATE VIEW problem_attachments AS 
    SELECT  atts.id AS id, 
            atts.created_at AS created_at, 
            atts.last_updated_at AS last_updated_at, 
            atts.last_updated_by AS last_updated_by, 
            atts.visible AS visible, 
            atts.private AS private,
            atts.execable AS execable,
            atts.name AS name,
            atts.data AS data,
            atts.data_size AS data_size,
            pam.problem_id AS problem_id 
    FROM attachments atts, problem_attachments_m2m pam 
    WHERE atts.id = pam.attachment_id;


COMMIT;