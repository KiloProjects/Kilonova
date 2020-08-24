package proto

// This file specifies the message type stuff

type Compile struct {
	Code     string `json:"code"`
	Language string `json:"lang"`
	ID       int    `json:"id"`
}

type CResponse struct {
	ID      int    `json:"id"`
	Output  string `json:"output"`
	Success bool   `json:"success"`
	Other   string `json:"other"`
}

type STask struct {
	ID          int     `json:"id"`
	TID         int     `json:"tid"`
	Language    string  `json:"language"`
	Input       string  `json:"input"`
	Filename    string  `json:"filename"`
	TimeLimit   float64 `json:"timelimit"`
	MemoryLimit int     `json:"memorylimit"`
	StackLimit  int     `json:"stacklimit"`
}

type STResponse struct {
	TID      int     `json:"tid"`
	Output   string  `json:"output"`
	Time     float64 `json:"time"`
	Memory   int     `json:"memory"`
	Status   int     `json:"int"`
	Comments string  `json:"comments"`
}

type TRemove struct {
	ID int `json:"id"`
}

type Error struct {
	Value string `json:"value"`
}

// possible extensions

type Assign struct {
	Count int `json:"count"`
}

type QLen struct {
	Length int `json:"length"`
}
