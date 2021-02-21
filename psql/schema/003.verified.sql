ALTER TABLE users ADD COLUMN verified_email bool NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN email_verif_sent_at timestamptz;
