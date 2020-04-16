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
	Name        string  `json:"name"`
	Text        string  `json:"text"`
	Tests       []Test  `json:"tests"`
	TestName    string  `json:"testName"`
	Author      User    `json:"author"`
	AuthorID    int     `json:"author_id"`
	MemoryLimit float32 `json:"memoryLimit"`
	StackLimit  float32 `json:"stackLimit"`
	SourceSize  int     `json:"sourceSize"`
}

// Test is the type for sample test
type Test struct {
	gorm.Model
	Score     int `json:"score"`
	ProblemID int `json:"problemid"`
}

// EvalTest is the type for tests meant for evaluation
type EvalTest struct {
	gorm.Model
	TestID int  `json:"testid"`
	Test   Test `json:"test"`
	UserID int  `json:"userid"`
	User   User `json:"user"`
	TaskID int  `json:"taskid"`
}

// Task is a source
type Task struct {
	gorm.Model
	SourceCode string     `json:"code,omitempty"`
	Type       int        `json:"type"`
	UserID     int        `json:"userid"`
	User       User       `json:"user"`
	Score      *int       `json:"score"`
	Tests      []EvalTest `json:"tests"`
	Problem    Problem    `json:"problem"`
}
