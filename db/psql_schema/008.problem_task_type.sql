CREATE TYPE problem_task_type AS ENUM ('batch', 'communication');

ALTER TABLE problems ADD COLUMN task_type problem_task_type NOT NULL DEFAULT 'batch';
ALTER TABLE problems ADD COLUMN communication_num_processes integer NOT NULL DEFAULT 1;
