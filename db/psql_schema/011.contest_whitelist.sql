DROP TABLE IF EXISTS contest_limits;

ALTER TABLE contests ADD COLUMN ip_management_enabled BOOLEAN DEFAULT false;
ALTER TABLE contests ADD COLUMN whitelist_enabled BOOLEAN DEFAULT false;