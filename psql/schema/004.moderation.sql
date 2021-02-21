ALTER TABLE users ADD COLUMN banned boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN disabled boolean NOT NULL DEFAULT false;
