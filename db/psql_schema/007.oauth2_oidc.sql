
CREATE TABLE IF NOT EXISTS oauth_clients (
    id uuid PRIMARY KEY,
    allowed_redirects text[] NOT NULL,
    allowed_post_logout_redirects text[] NOT NULL,
    app_type integer NOT NULL,
    developer_mode boolean NOT NULL,

    secret_hash text,

    created_at timestamp NOT NULL DEFAULT now(),
    author_id bigint REFERENCES users(id) ON DELETE SET NULL,

    name text NOT NULL
);

CREATE TABLE IF NOT EXISTS oauth_requests (
    id uuid PRIMARY KEY,
    created_at timestamp NOT NULL DEFAULT NOW(),
    auth_time timestamp,
    application_id uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    callback_uri text NOT NULL,
    user_id bigint REFERENCES users(id) ON DELETE CASCADE,
    scopes text[] NOT NULL,
    response_type text NOT NULL,
    response_mode text NOT NULL,
    transfer_state text NOT NULL,
    nonce text,
    code_challenge text,
    code_challenge_method text,
    prompt text[],
    ui_locales text[],
    login_hint text,
    expires_at timestamp,
    request_done boolean NOT NULL DEFAULT FALSE,
    request_code text
);

CREATE TYPE oauth_token_type AS ENUM ('access', 'refresh');

CREATE TABLE IF NOT EXISTS oauth_tokens (
    id uuid PRIMARY KEY,
    created_at timestamp NOT NULL DEFAULT NOW(),
    expires_at timestamp NOT NULL,
    application_id uuid REFERENCES oauth_clients(id) ON DELETE CASCADE,
    user_id bigint REFERENCES users(id) ON DELETE CASCADE,
    token_type oauth_token_type NOT NULL,
    scopes text[] NOT NULL,
    audience text[] NOT NULL,
    -- from_token is used to track the token that was used to create this specific token
    -- if the token is revoked, this token is also revoked
    from_token uuid REFERENCES oauth_tokens(id) ON DELETE CASCADE,

    amr text[],
    auth_time timestamp
);