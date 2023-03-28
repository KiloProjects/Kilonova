INSERT INTO attachments 
    (problem_id, visible, private, name, data) 
    SELECT id, false, false, 'statement-ro.md', convert_to(description, 'utf-8') 
        FROM problems;

ALTER TABLE problems DROP COLUMN description;
ALTER TABLE problems DROP COLUMN short_description;

