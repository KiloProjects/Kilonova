CREATE TYPE leaderboard_type AS enum (
    'classic',
    'acm-icpc'
);

ALTER TABLE contests ADD COLUMN leaderboard_style leaderboard_type NOT NULL DEFAULT 'classic';
ALTER TABLE contests ADD COLUMN leaderboard_freeze_time timestamptz DEFAULT NULL;
ALTER TABLE contests ADD COLUMN icpc_submission_penalty integer NOT NULL DEFAULT 20;
