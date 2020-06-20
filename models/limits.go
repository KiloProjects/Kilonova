package models

// Limits stores the constraints that need to be respected by a task
type Limits struct {
	// seconds
	TimeLimit float64 `json:"timeLimit"`
	// kilobytes
	StackLimit  int `json:"stackLimit"`
	MemoryLimit int `json:"memoryLimit"`
}
