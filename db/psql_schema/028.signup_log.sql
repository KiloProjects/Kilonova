
CREATE TABLE IF NOT EXISTS signup_logs (
    created_at timestamptz NOT NULL DEFAULT NOW(),
    
    user_id bigint NOT NULL,
    ip_addr inet,
    user_agent text
);
