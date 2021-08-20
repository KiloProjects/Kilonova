package kilonova

import (
	"time"
)

type ProblemType string

const (
	ProblemTypeNone          ProblemType = ""
	ProblemTypeClassic       ProblemType = "classic"
	ProblemTypeCustomChecker ProblemType = "custom_checker"
	// TODO
	ProblemTypeInteractive ProblemType = "interactive"
)

var (
	ErrAttachmentExists = &Error{Code: EINVALID, Message: "Attachment with that name already exists!"}
)

type Problem struct {
	ID            int       `json:"id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ShortDesc     string    `json:"short_description" db:"short_description"`
	TestName      string    `json:"test_name" db:"test_name"`
	AuthorID      int       `json:"author_id" db:"author_id"`
	Visible       bool      `json:"visible"`
	DefaultPoints int       `json:"default_points" db:"default_points"`

	// Limit stuff
	TimeLimit   float64 `json:"time_limit" db:"time_limit"`
	MemoryLimit int     `json:"memory_limit" db:"memory_limit"`
	StackLimit  int     `json:"stack_limit" db:"stack_limit"`
	SourceSize  int     `json:"source_size" db:"source_size"`

	SourceCredits string `json:"source_credits" db:"source_credits"`
	AuthorCredits string `json:"author_credits" db:"author_credits"`

	// Eval stuff
	Type         ProblemType `json:"type" db:"pb_type"`
	ConsoleInput bool        `json:"console_input" db:"console_input"`
}

// ProblemFilter is the struct with all filterable fields on the problem
// It also provides a Limit and Offset field, for pagination
// This list might be expanded as time goes on
type ProblemFilter struct {
	ID           *int    `json:"id"`
	IDs          []int   `json:"ids"`
	AuthorID     *int    `json:"author_id"`
	ConsoleInput *bool   `json:"console_input"`
	Visible      *bool   `json:"visible"`
	Name         *string `json:"name"`

	LookingUserID *int `json:"looking_user_id"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ProblemUpdate struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	ShortDesc     *string `json:"short_desc"`
	TestName      *string `json:"test_name"`
	AuthorID      *int    `json:"author_id"`
	DefaultPoints *int    `json:"default_points"`

	TimeLimit   *float64 `json:"time_limit"`
	MemoryLimit *int     `json:"memory_limit"`
	StackLimit  *int     `json:"stack_limit"`
	SourceSize  *int     `json:"source_size"`

	SourceCredits *string `json:"source_credits"`
	AuthorCredits *string `json:"author_credits"`

	Type          ProblemType `json:"type"`
	SubtaskString *string     `json:"subtask_string"`
	ConsoleInput  *bool       `json:"console_input"`
	Visible       *bool       `json:"visible"`
}

type Attachment struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ProblemID int       `json:"problem_id" db:"problem_id"`
	Visible   bool      `json:"visible"`
	Private   bool      `json:"private"`

	Name string `json:"name"`
	Data []byte `json:"data,omitempty"`
	Size int    `json:"data_size" db:"data_size"`
}

type AttachmentFilter struct {
	ID        *int    `json:"id"`
	ProblemID *int    `json:"problem_id"`
	Visible   *bool   `json:"visible"`
	Private   *bool   `json:"private"`
	Name      *string `json:"name"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type AttachmentUpdate struct {
	Visible *bool   `json:"visible"`
	Private *bool   `json:"private"`
	Name    *string `json:"name"`
}
