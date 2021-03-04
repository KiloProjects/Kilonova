package kilonova

import (
	"context"
	"io"

	"github.com/KiloProjects/kilonova/internal/config"
)

// EVERYTHING IS A TODO HERE AND WILL CHANGE ACCORDING TO PLAN

type Sandbox interface {
	ReadFile(path string) (io.ReadSeekCloser, error)
	WriteFile(path string, r io.Reader) error
	FileExists(path string) bool

	// TODO: Delete
	CopyFromBox(p1 string, w io.Writer) error
	CopyInBox(p1, p2 string) error

	// if stdout == stderr, then it will act like exec.CombinedOutput()
	RunCommand(ctx context.Context, cmd []string, conf *RunConfig) (*RunStats, error)

	Close() error
}

type Checker interface {
	RunChecker(programOut io.Reader, correctOut io.Reader, maxScore int) (string, int)
}

type Runner interface {
	Compile(ctx context.Context, cr *CompileRequest) (*CompileResponse, error)
	Execute(ctx context.Context, er *ExecRequest) (*ExecResponse, error)
	Clean(ctx context.Context, subid int) error
	Close(ctx context.Context) error
}

type CompileRequest struct {
	ID   int
	Code []byte
	Lang string
}

type CompileResponse struct {
	Output  string
	Success bool
	Other   string
}

// TODO

/*
	protobufTest := &kilonova.ExecRequest{
		ID:          int32(sub.ID),
		TID:         int32(test.ID),
		Filename:    filename,
		StackLimit:  int32(problem.StackLimit),
		MemoryLimit: int32(problem.MemoryLimit),
		TimeLimit:   problem.TimeLimit,
		Lang:        sub.Language,
		TestID:      int64(pbTest.ID),
	}
*/
type ExecRequest struct {
	SubID       int
	SubtestID   int
	TestID      int
	Filename    string
	StackLimit  int
	MemoryLimit int
	TimeLimit   float64
	Lang        string
}

type ExecResponse struct {
	SubtestID  int
	Time       float64
	Memory     int
	ExitStatus int
	Comments   string
}

type RunConfig struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	InputPath  string
	OutputPath string

	MemoryLimit int
	StackLimit  int

	TimeLimit      float64
	WallTimeLimit  float64
	ExtraTimeLimit float64

	MaxProcs int

	InheritEnv   bool
	EnvToInherit []string
	EnvToSet     map[string]string

	Directories []config.Directory
}

type RunStats struct {
	Memory int `json:"memory"`

	ExitCode   int  `json:"exit_code"`
	ExitSignal int  `json:"exit_signal"`
	Killed     bool `json:"killed"`

	Message string `json:"message"`
	Status  string `json:"status"`

	Time     float64 `json:"time"`
	WallTime float64 `json:"wall_time"`
}

type Grader interface {
	Compile(code string, lang string) error
}

type FullProblem struct {
	Problem
	Tests []*FullTest
}

type FullTest struct {
	Test
	Input  io.ReadCloser
	Output io.ReadCloser
}
