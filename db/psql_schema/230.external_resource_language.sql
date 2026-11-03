UPDATE external_resources SET language = '' WHERE language IS NULL;
ALTER TABLE external_resources ALTER COLUMN language SET DEFAULT '';
ALTER TABLE external_resources ALTER COLUMN language SET NOT NULL;
