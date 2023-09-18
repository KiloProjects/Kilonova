
CREATE INDEX IF NOT EXISTS user_admins ON users (admin) WHERE admin = true;
