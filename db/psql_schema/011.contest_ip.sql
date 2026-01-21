CREATE TABLE IF NOT EXISTS contest_limits
(
    contest_id               integer NOT NULL REFERENCES contests (id) ON DELETE CASCADE ON UPDATE CASCADE,
    -- ip_management_enabled is ONLY for the IP management UI. It does not affect the actual IP matching.
    ip_management_enabled    boolean NOT NULL DEFAULT false,
    whitelisting_enabled     boolean NOT NULL DEFAULT false,
    past_submissions_enabled boolean NOT NULL DEFAULT true
);