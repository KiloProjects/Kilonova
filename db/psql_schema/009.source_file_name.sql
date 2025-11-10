ALTER TABLE submissions ADD COLUMN code_filename TEXT;
-- path.Base() of SourceName
UPDATE submissions SET code_filename = 'main.cpp' WHERE language IN ('cpp11', 'cpp14', 'cpp17', 'cpp20');
UPDATE submissions SET code_filename = 'main.c' WHERE language = 'c';
UPDATE submissions SET code_filename = 'main.pas' WHERE language = 'pascal';
UPDATE submissions SET code_filename = 'main.go' WHERE language = 'golang';
UPDATE submissions SET code_filename = 'main.hs' WHERE language = 'haskell';
UPDATE submissions SET code_filename = 'Main.java' WHERE language = 'java';
UPDATE submissions SET code_filename = 'main.kt' WHERE language = 'kotlin';
UPDATE submissions SET code_filename = 'main.py' WHERE language = 'python3';
UPDATE submissions SET code_filename = 'index.js' WHERE language = 'nodejs';
UPDATE submissions SET code_filename = 'index.php' WHERE language = 'php';
UPDATE submissions SET code_filename = 'main.rs' WHERE language = 'rust';
UPDATE submissions SET code_filename = 'main.txt' WHERE language = 'outputOnly';

ALTER TABLE submissions ALTER COLUMN code_filename SET NOT NULL;