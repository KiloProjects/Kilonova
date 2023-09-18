ALTER TABLE submission_subtasks ADD COLUMN computed_score NUMERIC NOT NULL GENERATED ALWAYS AS (ROUND(COALESCE(final_percentage, 0) * (score / 100.0), digit_precision)) STORED;

