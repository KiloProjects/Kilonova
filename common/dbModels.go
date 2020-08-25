package common

import "gorm.io/gorm"

const (
	_ = iota
	cpp
	c
	golang
	python
)

// User represents a user profile
type User struct {
	gorm.Model
	Name     string `json:"name"`
	Admin    bool   `json:"isAdmin,omitempty"`
	Proposer bool   `json:"isProposer,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"-"`
}

// Problem is the main object for problem
type Problem struct {
	gorm.Model
	Name     string `json:"title"`
	Text     string `json:"description"`
	Tests    []Test `json:"tests,omitempty"`
	TestName string `json:"testName"`
	// User is the author
	User   User `json:"author"`
	UserID uint `json:"-"`
	/// LIMITS
	// seconds
	TimeLimit float64 `json:"timeLimit"`
	// kilobytes
	StackLimit  uint64 `json:"stackLimit"`
	MemoryLimit uint64 `json:"memoryLimit"`
	/// /LIMITS
	SourceSize   int64 `json:"sourceSize"`
	ConsoleInput bool  `json:"consoleInput"`
	Visible      bool  `json:"visible"`
}

// Test is the type for sample test
// NOTE: When Score < 0, it means that an error occurred
type Test struct {
	gorm.Model
	Score     int  `json:"score"`
	ProblemID uint `json:"problemID"`
	VisibleID uint `json:"visibleID"`
}

// EvalTest is the type for tests meant for evaluation
// NOTE: When Score < 0, it means that an error occurred
type EvalTest struct {
	gorm.Model
	Done bool `json:"done"`

	// Output is the text displayed on the frontend (like `Fatal signal 11` or `Missing output file`)
	Output string  `json:"resultinfo"`
	Time   float64 `json:"timeTaken"`
	Memory int     `json:"memoryUsed"`
	Score  int     `json:"score"`
	Test   Test    `json:"test"`
	TestID uint    `json:"testID"`
	UserID uint    `json:"userID"`
	TaskID uint    `json:"taskID"`
}

// Task is the type for user-submitted tasks
type Task struct {
	gorm.Model
	SourceCode     string     `json:"code,omitempty"`
	User           User       `json:"user"`
	UserID         uint       `json:"userid"`
	Tests          []EvalTest `json:"tests,omitempty"`
	Problem        Problem    `json:"problem"`
	ProblemID      uint       `json:"problemid"`
	Language       string     `json:"language"`
	Status         int        `json:"status"`
	CompileError   bool       `json:"compileError"`
	CompileMessage string     `json:"compileMessage"`
	Score          int        `json:"score"`

	// Visible controls the visibility of source code of the task to non-admin and not-author users
	Visible bool `json:"visible"`
}

const (
	// These represent the different possible statuses of a task

	// StatusWaiting is the initial state, the Task hasn't been picked up yet
	StatusWaiting = iota
	// StatusWorking is the state when a Task has been picked up by a box but hasn't yet finished
	StatusWorking
	// StatusDone is the state when a Task has been fully graded
	StatusDone
)
