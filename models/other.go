package models

import "github.com/jinzhu/gorm"

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
