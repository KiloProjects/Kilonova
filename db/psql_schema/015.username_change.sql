CREATE TABLE IF NOT EXISTS username_change_history (
    user_id     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    changed_at  timestamptz NOT NULL,
    name    text        NOT NULL
);

CREATE INDEX username_change_history_uid ON username_change_history(user_id);

CREATE OR REPLACE FUNCTION insert_username_change() RETURNS TRIGGER AS $$
    BEGIN
        IF OLD.name IS DISTINCT FROM NEW.name THEN
            INSERT INTO username_change_history (user_id, changed_at, name) VALUES (NEW.id, NOW(), NEW.name); 
        END IF;
        RETURN NULL;
    END;
$$ LANGUAGE plpgsql;

INSERT INTO username_change_history SELECT id AS user_id, created_at AS changed_at, name AS name FROM users;

CREATE OR REPLACE TRIGGER username_change_inserts 
    AFTER INSERT OR UPDATE OF name 
    ON users
    FOR EACH ROW
    EXECUTE FUNCTION insert_username_change();


ALTER TABLE users ADD COLUMN name_change_required boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN locked_login boolean NOT NULL DEFAULT false;
