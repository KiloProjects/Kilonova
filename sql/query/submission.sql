-- name: Submission :one
SELECT * FROM submissions
WHERE id = $1;

-- name: SubTests :many
SELECT * FROM submission_tests 
WHERE submission_id = $1;

-- name: CreateSubmission :one
INSERT INTO submissions (
	user_id, problem_id, code, language
) VALUES (
	$1, $2, $3, $4
) RETURNING id;

-- name: CreateSubTest :exec
INSERT INTO submission_tests (
	user_id, test_id, submission_id
) VALUES (
	$1, $2, $3
);

-- name: UserProblemSubmissions :many
SELECT * FROM submissions 
WHERE user_id = $1 AND problem_id = $2;

-- name: MaxScore :one
SELECT score FROM submissions 
WHERE user_id = $1 AND problem_id = $2 
ORDER BY score desc 
LIMIT 1;

-- name: Submissions :many
SELECT * FROM submissions;

-- name: WaitingSubmissions :many
SELECT * FROM submissions
WHERE status = 'waiting';

-- name: SetSubmissionVisibility :exec
UPDATE submissions
SET visible = $2
WHERE id = $1;

-- name: SetCompilation :exec
UPDATE submissions
SET compile_error = $2, compile_message = $3 
WHERE id = $1;

-- name: SetSubmissionStatus :exec
UPDATE submissions
SET status = $2, score = $3
WHERE id = $1;

-- name: SetSubmissionTest :exec
UPDATE submission_tests 
SET verdict = $2, time = $3, memory = $4, score = $5, done = true
WHERE id = $1;
