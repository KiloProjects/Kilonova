package models

import "github.com/jinzhu/gorm"

const (
	_ = iota
	cpp
	c
	golang
	python
)

// KNContextType is the string type for all context values
type KNContextType string

// Config is the main information for the platform
type Config struct {
	SecretKey string `json:"secretKey"`
}

// Session represents the data storred in a session cookie
type Session struct {
	IsAdmin bool `json:"isAdmin"`
	UserID  uint `json:"userID"`
}

// MOTD presents a random message of the day
type MOTD struct {
	gorm.Model
	Motd string `json:"motd,omitempty"`
}

// User represents a user profile
type User struct {
	gorm.Model
	Name           string    `json:"name"`
	IsAdmin        bool      `json:"isAdmin,omitempty"`
	Email          string    `json:"email,omitempty"`
	SolvedProblems []Problem `json:"solvedProblems"`
	Tasks          []Task    `json:"sentTasks"`
	Password       string    `json:"-"`
}

// Problem is the main object for problem
type Problem struct {
	gorm.Model
	Name     string `json:"title"`
	Text     string `json:"description"`
	Tests    []Test `json:"tests"`
	TestName string `json:"testName"`
	// User is the author
	User       User   `json:"author"`
	UserID     uint   `json:"author_id"`
	Limits     Limits `json:"limits"`
	SourceSize int    `json:"sourceSize"`
}

// Test is the type for sample test
type Test struct {
	gorm.Model
	Score     int  `json:"score"`
	ProblemID uint `json:"problemID"`
}

// EvalTest is the type for tests meant for evaluation
type EvalTest struct {
	gorm.Model
	Done bool `json:"done"`

	Test   Test `json:"test"`
	TestID uint `json:"testID"`

	User   User `json:"user"`
	UserID uint `json:"userID"`

	Task   Task `json:"task"`
	TaskID uint `json:"taskID"`
}

// Task is the type for user-submitted tasks
type Task struct {
	gorm.Model
	SourceCode string     `json:"code,omitempty"`
	Type       int        `json:"type"`
	User       User       `json:"user"`
	UserID     uint       `json:"userid"`
	Score      *int       `json:"score"`
	Tests      []EvalTest `json:"tests"`
	Problem    Problem    `json:"problem"`
	ProblemID  uint       `json:"problemid"`
	Language   string
	Status     int
}

const (
	_ = iota
	// StatusWaiting is the initial state, the Task hasn't been picked up yet
	StatusWaiting
	// StatusWorking is the state when a Task has been picked up by a box but hasn't yet finished
	StatusWorking
	// StatusDone is the state when a Task has been fully graded
	StatusDone
)
