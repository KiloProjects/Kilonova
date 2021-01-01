-- name: CreateUser :one
INSERT INTO users (
	email, name, password
) VALUES (
	$1, $2, $3
) 
RETURNING *;

-- name: User :one
SELECT * FROM users 
WHERE id = $1;

-- name: UserByEmail :one
SELECT * FROM users 
WHERE lower(email) = lower(sqlc.arg(email));

-- name: UserByName :one 
SELECT * FROM users 
WHERE lower(name) = lower(sqlc.arg(username));

-- name: CountUsers :one
SELECT COUNT(*) FROM users 
WHERE lower(name) = lower(sqlc.arg(username)) OR lower(email) = lower(sqlc.arg(email));

-- name: Users :many
SELECT * FROM users
ORDER BY id;

-- name: Admins :many
SELECT * FROM users
WHERE admin = true
ORDER BY id;

-- name: Proposers :many
SELECT * FROM users 
WHERE proposer = true OR admin = true
ORDER BY id;


-- name: Top100 :many
-- I am extremely proud of this
SELECT us.*, COUNT(sub.user_id) AS number_problems
FROM users us
LEFT JOIN (
	SELECT problem_id, user_id
	FROM submissions 
	WHERE score = 100 
	GROUP BY problem_id, user_id
) sub
ON   sub.user_id = us.id
GROUP BY us.id 
ORDER BY COUNT(sub.user_id) desc, us.id 
LIMIT 100;

-- name: SolvedProblems :many
SELECT problem_id FROM submissions
WHERE score = 100 AND user_id = $1
GROUP BY problem_id
ORDER BY problem_id;

-- name: SetProposer :exec
UPDATE users SET proposer = $2
WHERE id = $1;

-- name: SetAdmin :exec
UPDATE users SET admin = $2
WHERE id = $1;


-- name: SetBio :exec
UPDATE users SET bio = $2
WHERE id = $1;

-- name: SetDefaultVisibility :exec
UPDATE users SET default_visible = $2
WHERE id = $1;

-- name: SetEmail :exec
UPDATE users SET email = $2
WHERE id = $1;

-- name: SetPassword :exec
UPDATE users SET password = $2
WHERE id = $1;

-- name: SetVerification :exec
UPDATE users SET verified_email = $2
WHERE id = $1;
