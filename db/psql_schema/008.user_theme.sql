
CREATE TYPE IF NOT EXISTS theme_type AS enum (
    'light',
    'dark'
);

-- dark looks better.
ALTER TABLE users ADD COLUMN preferred_theme theme_type NOT NULL DEFAULT 'dark'; 

