

ALTER TABLE session_clients DROP CONSTRAINT session_clients_session_id_fkey;
ALTER TABLE session_clients ADD COLUMN user_id bigint;
UPDATE session_clients SET user_id = (SELECT user_id FROM sessions WHERE id = session_id LIMIT 1);
