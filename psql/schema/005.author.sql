
ALTER TABLE problems RENAME COLUMN credits TO source_credits;
ALTER TABLE problems ADD COLUMN author_credits text NOT NULL DEFAULT '';

-- TODO: Use this sometime
ALTER TABLE problems ADD COLUMN short_description text NOT NULL DEFAULT '';
