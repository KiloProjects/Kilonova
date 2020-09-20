-- name: CreateUser :one
INSERT INTO users (
	email, name, password
) VALUES (
	$1, $2, $3
) 
RETURNING id;

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
SELECT * FROM users;

-- name: Admins :many
SELECT * FROM users
WHERE admin = true;

-- name: Proposers :many
SELECT * FROM users 
WHERE proposer = true;



-- name: SetProposer :exec
UPDATE users SET proposer = $2
WHERE id = $1;

-- name: SetAdmin :exec
UPDATE users SET admin = $2
WHERE id = $1;


-- name: SetBio :exec
UPDATE users SET bio = $2
WHERE id = $1;


-- name: SetEmail :exec
UPDATE users SET email = $2
WHERE id = $1
RETURNING *;
