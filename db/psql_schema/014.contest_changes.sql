
ALTER TABLE contests ADD COLUMN virtual boolean NOT NULL DEFAULT false;
ALTER TABLE contests DROP COLUMN hidden;
ALTER TABLE contests ADD COLUMN visible boolean NOT NULL DEFAULT false;