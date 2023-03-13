

ALTER TABLE attachments ADD COLUMN execable boolean NOT NULL DEFAULT FALSE;

-- Heuristic based on the attachments from the main instance.
UPDATE attachments SET execable = true WHERE 
    name = '.output_only' OR (
        lower(name) NOT LIKE '\_%' AND -- exclude prefixed with `_`
        lower(name) NOT LIKE '%.png' AND 
        lower(name) NOT LIKE '%.jpg' AND 
        lower(name) NOT LIKE '%.webp' AND 
        lower(name) NOT LIKE '%.pdf' AND 
        lower(name) NOT LIKE '%.zip' AND 
        lower(name) NOT LIKE '%.md' AND 
        lower(name) NOT LIKE '%.txt'
    );