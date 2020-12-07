
-- name: Test :one
SELECT * FROM tests 
WHERE id = $1 AND orphaned = false;

-- name: TestVisibleID :one
SELECT * FROM tests 
WHERE problem_id = $1 AND visible_id = $2 AND orphaned = false
ORDER BY visible_id;

-- name: CreateTest :one
INSERT INTO tests (
	problem_id, visible_id, score 
) VALUES (
	$1, $2, $3
) RETURNING *;

-- name: SetVisibleID :exec
UPDATE tests 
SET visible_id = $2
WHERE id = $1 AND orphaned = false;

-- name: BiggestVID :one
SELECT visible_id 
FROM tests 
WHERE problem_id = $1 AND orphaned = false
ORDER BY visible_id desc
LIMIT 1;

-- name: SetPbTestVisibleID :exec
UPDATE tests 
SET visible_id = sqlc.arg(new_id)
WHERE problem_id = sqlc.arg(problem_id) AND visible_id = sqlc.arg(old_id) AND orphaned = false;

-- name: SetPbTestScore :exec
UPDATE tests 
SET score = $3
WHERE problem_id = $1 AND visible_id = $2 AND orphaned = false;
