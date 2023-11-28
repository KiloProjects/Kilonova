
CREATE TABLE IF NOT EXISTS contest_invitations (
    id          text    PRIMARY KEY,
    created_at  timestamptz NOT NULL DEFAULT NOW(),
    contest_id  integer     NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    creator_id  integer     REFERENCES users(id) ON DELETE SET NULL,
    expired     boolean     NOT NULL DEFAULT FALSE
);

ALTER TABLE contest_registrations ADD COLUMN invitation_id text REFERENCES contest_invitations(id) ON DELETE SET NULL;
