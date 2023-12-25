
-- Just official and virtual to start with. More may be created later
CREATE TYPE contest_type AS ENUM (
    'official',
    'virtual'
);

-- Initially make existing contests official
ALTER TABLE contests ADD COLUMN type contest_type NOT NULL DEFAULT 'official';

-- Then, newer ones are classified as virtual at the beginning
ALTER TABLE contests ALTER COLUMN type SET DEFAULT 'virtual';
