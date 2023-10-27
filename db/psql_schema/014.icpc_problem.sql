
ALTER TYPE scoring_type ADD VALUE 'acm-icpc';

CREATE TYPE eval_type AS enum (
    'classic',
    'acm-icpc'
);

ALTER TABLE submissions ADD COLUMN submission_type eval_type NOT NULL DEFAULT 'classic';
ALTER TABLE submissions ADD COLUMN icpc_verdict text;
