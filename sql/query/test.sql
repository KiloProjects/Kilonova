
-- name: Test :one
SELECT * FROM tests 
WHERE id = $1;

-- name: TestVisibleID :one
SELECT * FROM tests 
WHERE problem_id = $1 AND visible_id = $2;

-- name: CreateTest :exec
INSERT INTO tests (
	problem_id, visible_id, score 
) VALUES (
	$1, $2, $3
);

-- name: SetVisibleID :exec
UPDATE tests 
SET visible_id = $2
WHERE id = $1;

-- name: BiggestVID :one
SELECT visible_id 
FROM tests 
WHERE problem_id = $1
ORDER BY visible_id desc
LIMIT 1;

-- name: SetPbTestVisibleID :exec
UPDATE tests 
SET visible_id = sqlc.arg(new_id)
WHERE problem_id = sqlc.arg(problem_id) AND visible_id = sqlc.arg(old_id);

-- name: SetPbTestScore :exec
UPDATE tests 
SET score = $3
WHERE problem_id = $1 AND visible_id = $2;
