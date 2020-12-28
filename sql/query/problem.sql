-- name: Problem :one
SELECT * FROM problems
WHERE id = $1;

-- name: Problems :many
SELECT * FROM problems
ORDER BY id;

-- name: ProblemTests :many
SELECT * FROM tests 
WHERE problem_id = $1 AND orphaned = false
ORDER BY visible_id;

-- name: VisibleProblems :many
SELECT * FROM problems 
WHERE visible = true OR author_id = $1
ORDER BY id;

-- name: CreateProblem :one
INSERT INTO problems (
	name, author_id, console_input, test_name, memory_limit, stack_limit, time_limit 
) VALUES (
	$1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: SetLimits :exec
UPDATE problems 
SET memory_limit = $2, stack_limit = $3, time_limit = $4
WHERE id = $1;

-- name: ProblemByName :one
SELECT * FROM problems
WHERE lower(name) = lower(sqlc.arg(name));

-- name: SetTimeLimit :exec
UPDATE problems
SET time_limit = $2
WHERE id = $1;

-- name: SetMemoryLimit :exec
UPDATE problems
SET memory_limit = $2
WHERE id = $1;

-- name: SetStackLimit :exec
UPDATE problems
SET stack_limit = $2
WHERE id = $1;

-- name: SetPbCredits :exec
UPDATE problems
SET credits = $2
WHERE id = $1;

-- name: SetPbVisibility :exec
UPDATE problems 
SET visible = $2
WHERE id = $1;

-- name: SetPbName :exec
UPDATE problems 
SET name = $2 
WHERE id = $1;

-- name: SetPbDescription :exec
UPDATE problems 
SET description = $2
WHERE id = $1;

-- name: SetConsoleInput :exec
UPDATE problems 
SET console_input = $2
WHERE id = $1;

-- name: SetTestName :exec
UPDATE problems 
SET test_name = $2
WHERE id = $1;

-- name: PurgePbTests :exec
-- Since there is a key constraint on these tests, instead of removing them, we simply orphan them so they don't get used in the future
UPDATE tests 
SET orphaned = true
WHERE problem_id = $1;
