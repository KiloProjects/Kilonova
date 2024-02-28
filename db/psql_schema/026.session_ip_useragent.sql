
CREATE TABLE IF NOT EXISTS session_clients (
    session_id text NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    last_checked_at timestamptz NOT NULL DEFAULT NOW(),

    ip_addr inet,
    user_agent text,

    CONSTRAINT unique_client_tuple UNIQUE (session_id, ip_addr, user_agent)
);

