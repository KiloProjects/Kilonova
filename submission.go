package kilonova

import (
	"database/sql"
	"fmt"
	"time"
)

type Status string

const (
	StatusNone     Status = ""
	StatusCreating Status = "creating"
	StatusWaiting  Status = "waiting"
	StatusWorking  Status = "working"
	StatusFinished Status = "finished"
)

type Submission struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UserID    int       `db:"user_id" json:"user_id"`
	ProblemID int       `db:"problem_id" json:"problem_id"`
	Language  string    `json:"language"`
	Code      string    `json:"code,omitempty"`
	Status    Status    `json:"status"`

	CompileError   sql.NullBool   `db:"compile_error" json:"compile_error"`
	CompileMessage sql.NullString `json:"compile_message,omitempty" db:"compile_message"`

	MaxTime   float64 `json:"max_time" db:"max_time"`
	MaxMemory int     `json:"max_memory" db:"max_memory"`

	Score int `json:"score"`

	// TODO: Remove after restarting
	// Visible bool `json:"visible"`
	// Quality bool `json:"quality"`
}

type SubmissionUpdate struct {
	Status Status
	Score  *int

	CompileError   *bool
	CompileMessage *string

	MaxTime   *float64
	MaxMemory *int
}

type SubmissionFilter struct {
	ID        *int `json:"id"`
	UserID    *int `json:"user_id"`
	ProblemID *int `json:"problem_id"`

	Status       Status  `json:"status"`
	Lang         *string `json:"lang"`
	Score        *int    `json:"score"`
	CompileError *bool   `json:"compile_error"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	Ordering  string `json:"ordering"`
	Ascending bool   `json:"ascending"`
}

type SubTest struct {
	ID           int       `json:"id"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	Done         bool      `json:"done"`
	Verdict      string    `json:"verdict"`
	Time         float64   `json:"time"`
	Memory       int       `json:"memory"`
	Score        int       `json:"score"`
	TestID       int       `db:"test_id" json:"test_id"`
	UserID       int       `db:"user_id" json:"user_id"`
	SubmissionID int       `db:"submission_id" json:"submission_id"`
}

type SubTestUpdate struct {
	Memory  *int
	Time    *float64
	Score   *int
	Verdict *string
	Done    *bool
}

// Utils for backends (I know, not so agnostic, but it makes life easier)

// Scan implements the sql.Scanner interface
func (e *Status) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = Status(s)
	case string:
		*e = Status(s)
	default:
		return fmt.Errorf("unsupported scan type for Status: %T", src)
	}
	return nil
}
