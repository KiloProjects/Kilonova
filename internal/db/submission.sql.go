// Code generated by sqlc. DO NOT EDIT.
// source: submission.sql

package db

import (
	"context"
	"database/sql"
)

const createSubTest = `-- name: CreateSubTest :exec
INSERT INTO submission_tests (
	user_id, test_id, submission_id
) VALUES (
	$1, $2, $3
)
`

type CreateSubTestParams struct {
	UserID       int64 `json:"user_id"`
	TestID       int64 `json:"test_id"`
	SubmissionID int64 `json:"submission_id"`
}

func (q *Queries) CreateSubTest(ctx context.Context, arg CreateSubTestParams) error {
	_, err := q.exec(ctx, q.createSubTestStmt, createSubTest, arg.UserID, arg.TestID, arg.SubmissionID)
	return err
}

const createSubmission = `-- name: CreateSubmission :one
INSERT INTO submissions (
	user_id, problem_id, code, language, visible
) VALUES (
	$1, $2, $3, $4, $5
) RETURNING id
`

type CreateSubmissionParams struct {
	UserID    int64  `json:"user_id"`
	ProblemID int64  `json:"problem_id"`
	Code      string `json:"code"`
	Language  string `json:"language"`
	Visible   bool   `json:"visible"`
}

func (q *Queries) CreateSubmission(ctx context.Context, arg CreateSubmissionParams) (int64, error) {
	row := q.queryRow(ctx, q.createSubmissionStmt, createSubmission,
		arg.UserID,
		arg.ProblemID,
		arg.Code,
		arg.Language,
		arg.Visible,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const maxScore = `-- name: MaxScore :one
SELECT score FROM submissions 
WHERE user_id = $1 AND problem_id = $2 
ORDER BY score desc 
LIMIT 1
`

type MaxScoreParams struct {
	UserID    int64 `json:"user_id"`
	ProblemID int64 `json:"problem_id"`
}

func (q *Queries) MaxScore(ctx context.Context, arg MaxScoreParams) (int32, error) {
	row := q.queryRow(ctx, q.maxScoreStmt, maxScore, arg.UserID, arg.ProblemID)
	var score int32
	err := row.Scan(&score)
	return score, err
}

const setCompilation = `-- name: SetCompilation :exec
UPDATE submissions
SET compile_error = $2, compile_message = $3 
WHERE id = $1
`

type SetCompilationParams struct {
	ID             int64          `json:"id"`
	CompileError   sql.NullBool   `json:"compile_error"`
	CompileMessage sql.NullString `json:"compile_message"`
}

func (q *Queries) SetCompilation(ctx context.Context, arg SetCompilationParams) error {
	_, err := q.exec(ctx, q.setCompilationStmt, setCompilation, arg.ID, arg.CompileError, arg.CompileMessage)
	return err
}

const setSubmissionStatus = `-- name: SetSubmissionStatus :exec
UPDATE submissions
SET status = $2, score = $3
WHERE id = $1
`

type SetSubmissionStatusParams struct {
	ID     int64  `json:"id"`
	Status Status `json:"status"`
	Score  int32  `json:"score"`
}

func (q *Queries) SetSubmissionStatus(ctx context.Context, arg SetSubmissionStatusParams) error {
	_, err := q.exec(ctx, q.setSubmissionStatusStmt, setSubmissionStatus, arg.ID, arg.Status, arg.Score)
	return err
}

const setSubmissionTest = `-- name: SetSubmissionTest :exec
UPDATE submission_tests 
SET verdict = $2, time = $3, memory = $4, score = $5, done = true
WHERE id = $1
`

type SetSubmissionTestParams struct {
	ID      int64   `json:"id"`
	Verdict string  `json:"verdict"`
	Time    float64 `json:"time"`
	Memory  int32   `json:"memory"`
	Score   int32   `json:"score"`
}

func (q *Queries) SetSubmissionTest(ctx context.Context, arg SetSubmissionTestParams) error {
	_, err := q.exec(ctx, q.setSubmissionTestStmt, setSubmissionTest,
		arg.ID,
		arg.Verdict,
		arg.Time,
		arg.Memory,
		arg.Score,
	)
	return err
}

const setSubmissionVisibility = `-- name: SetSubmissionVisibility :exec
UPDATE submissions
SET visible = $2
WHERE id = $1
`

type SetSubmissionVisibilityParams struct {
	ID      int64 `json:"id"`
	Visible bool  `json:"visible"`
}

func (q *Queries) SetSubmissionVisibility(ctx context.Context, arg SetSubmissionVisibilityParams) error {
	_, err := q.exec(ctx, q.setSubmissionVisibilityStmt, setSubmissionVisibility, arg.ID, arg.Visible)
	return err
}

const subTests = `-- name: SubTests :many
SELECT id, created_at, done, verdict, time, memory, score, test_id, user_id, submission_id FROM submission_tests 
WHERE submission_id = $1
ORDER BY id asc
`

func (q *Queries) SubTests(ctx context.Context, submissionID int64) ([]SubmissionTest, error) {
	rows, err := q.query(ctx, q.subTestsStmt, subTests, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SubmissionTest
	for rows.Next() {
		var i SubmissionTest
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.Done,
			&i.Verdict,
			&i.Time,
			&i.Memory,
			&i.Score,
			&i.TestID,
			&i.UserID,
			&i.SubmissionID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const submission = `-- name: Submission :one
SELECT id, created_at, user_id, problem_id, language, code, status, compile_error, compile_message, score, visible FROM submissions
WHERE id = $1
`

func (q *Queries) Submission(ctx context.Context, id int64) (Submission, error) {
	row := q.queryRow(ctx, q.submissionStmt, submission, id)
	var i Submission
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UserID,
		&i.ProblemID,
		&i.Language,
		&i.Code,
		&i.Status,
		&i.CompileError,
		&i.CompileMessage,
		&i.Score,
		&i.Visible,
	)
	return i, err
}

const submissions = `-- name: Submissions :many
SELECT id, created_at, user_id, problem_id, language, code, status, compile_error, compile_message, score, visible FROM submissions
ORDER BY id desc
`

func (q *Queries) Submissions(ctx context.Context) ([]Submission, error) {
	rows, err := q.query(ctx, q.submissionsStmt, submissions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Submission
	for rows.Next() {
		var i Submission
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UserID,
			&i.ProblemID,
			&i.Language,
			&i.Code,
			&i.Status,
			&i.CompileError,
			&i.CompileMessage,
			&i.Score,
			&i.Visible,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const userProblemSubmissions = `-- name: UserProblemSubmissions :many
SELECT id, created_at, user_id, problem_id, language, code, status, compile_error, compile_message, score, visible FROM submissions 
WHERE user_id = $1 AND problem_id = $2
ORDER BY id desc
`

type UserProblemSubmissionsParams struct {
	UserID    int64 `json:"user_id"`
	ProblemID int64 `json:"problem_id"`
}

func (q *Queries) UserProblemSubmissions(ctx context.Context, arg UserProblemSubmissionsParams) ([]Submission, error) {
	rows, err := q.query(ctx, q.userProblemSubmissionsStmt, userProblemSubmissions, arg.UserID, arg.ProblemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Submission
	for rows.Next() {
		var i Submission
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UserID,
			&i.ProblemID,
			&i.Language,
			&i.Code,
			&i.Status,
			&i.CompileError,
			&i.CompileMessage,
			&i.Score,
			&i.Visible,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const waitingSubmissions = `-- name: WaitingSubmissions :many
SELECT id, created_at, user_id, problem_id, language, code, status, compile_error, compile_message, score, visible FROM submissions
WHERE status = 'waiting'
ORDER BY id asc
`

func (q *Queries) WaitingSubmissions(ctx context.Context) ([]Submission, error) {
	rows, err := q.query(ctx, q.waitingSubmissionsStmt, waitingSubmissions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Submission
	for rows.Next() {
		var i Submission
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UserID,
			&i.ProblemID,
			&i.Language,
			&i.Code,
			&i.Status,
			&i.CompileError,
			&i.CompileMessage,
			&i.Score,
			&i.Visible,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
