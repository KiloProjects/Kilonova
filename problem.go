package kilonova

import (
	"context"
	"time"
)

type ProblemType string

const (
	ProblemTypeNone    ProblemType = ""
	ProblemTypeClassic ProblemType = "classic"
	// TODO
	ProblemTypeInteractive   ProblemType = "interactive"
	ProblemTypeCustomChecker ProblemType = "custom_checker"
)

type Problem struct {
	ID               int       `json:"id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	ShortDescription string    `json:"short_description" db:"short_description"`
	TestName         string    `json:"test_name" db:"test_name"`
	Author           *User     `json:"author,omitempty"`
	AuthorID         int       `json:"author_id" db:"author_id"`
	TimeLimit        float64   `json:"time_limit" db:"time_limit"`
	MemoryLimit      int       `json:"memory_limit" db:"memory_limit"`
	StackLimit       int       `json:"stack_limit" db:"stack_limit"`
	SourceSize       int       `json:"source_size" db:"source_size"`
	ConsoleInput     bool      `json:"console_input" db:"console_input"`
	Visible          bool      `json:"visible"`

	DefaultPoints int `json:"default_points" db:"default_points"`

	SourceCredits string `json:"source_credits" db:"source_credits"`
	AuthorCredits string `json:"author_credits" db:"author_credits"`

	Type           ProblemType `json:"type" db:"pb_type"`
	HelperCode     string      `json:"-" db:"helper_code"`
	HelperCodeLang string      `json:"-" db:"helper_code_lang"`
}

// ProblemFilter is the struct with all filterable fields on the problem
// It also provides a Limit and Offset field, for pagination
// This list might be expanded as time goes on
type ProblemFilter struct {
	ID           *int    `json:"id"`
	AuthorID     *int    `json:"author_id"`
	ConsoleInput *bool   `json:"console_input"`
	Visible      *bool   `json:"visible"`
	Name         *string `json:"name"`

	LookingUserID *int `json:"looking_user_id"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ProblemUpdate struct {
	Name         *string  `json:"name"`
	Description  *string  `json:"description"`
	ShortDesc    *string  `json:"short_desc"`
	TestName     *string  `json:"test_name"`
	AuthorID     *int     `json:"author_id"`
	TimeLimit    *float64 `json:"time_limit"`
	MemoryLimit  *int     `json:"memory_limit"`
	StackLimit   *int     `json:"stack_limit"`
	SourceSize   *int     `json:"source_size"`
	ConsoleInput *bool    `json:"console_input"`
	Visible      *bool    `json:"visible"`

	DefaultPoints *int `json:"default_points"`

	SourceCredits *string `json:"source_credits"`
	AuthorCredits *string `json:"author_credits"`

	Type           ProblemType `json:"type"`
	HelperCode     *string     `json:"helper_code"`
	HelperCodeLang *string     `json:"helper_code_lang"`
}

type ProblemService interface {
	ProblemByID(ctx context.Context, id int) (*Problem, error)

	Problems(ctx context.Context, filter ProblemFilter) ([]*Problem, error)

	CreateProblem(ctx context.Context, problem *Problem) error

	UpdateProblem(ctx context.Context, id int, upd ProblemUpdate) error

	DeleteProblem(ctx context.Context, id int) error
}
