CREATE TABLE IF NOT EXISTS audit_logs (
    id          bigserial 	    PRIMARY KEY,
    logged_at   timestamptz     NOT NULL DEFAULT NOW(),
    system_log  boolean         NOT NULL DEFAULT false,
    msg         text            NOT NULL DEFAULT '',
    author_id   bigint          DEFAULT null REFERENCES users(id) ON DELETE SET NULL
);