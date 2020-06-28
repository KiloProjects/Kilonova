// Package common contains stuff that can be used by all 4 components of the project (grader, API server, data manager and web UI)
package common

import "github.com/jinzhu/gorm"

// DataDir should be where most of the data is stored
const DataDir = "/data"

// Limits stores the constraints that need to be respected by a task
type Limits struct {
	// seconds
	TimeLimit float64 `json:"timeLimit"`
	// kilobytes
	StackLimit  int `json:"stackLimit"`
	MemoryLimit int `json:"memoryLimit"`
}

// Updater is an interface for a DB updater made by the boxManager (like updating the status of a task)
type Updater interface {
	Update(*gorm.DB) error
}
