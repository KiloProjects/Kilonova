

UPDATE attachments SET name = replace(name, 'checker', 'checker_legacy') WHERE name LIKE 'checker%';