
UPDATE problems SET source_size = GREATEST(30000, source_size);
ALTER TABLE problems ALTER COLUMN source_size SET DEFAULT 30000;
