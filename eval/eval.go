package eval

import (
	"context"
	"io"
	"io/fs"

	"github.com/KiloProjects/kilonova"
)

type Sandbox interface {
	ReadFile(path string) (io.ReadSeekCloser, error)
	WriteFile(path string, r io.Reader, mode fs.FileMode) error
	RemoveFile(path string) error
	FileExists(path string) bool

	GetID() int

	// if stdout == stderr, then it will act like exec.CombinedOutput()
	RunCommand(ctx context.Context, cmd []string, conf *RunConfig) (*RunStats, error)

	// Reset clears everything in the sandbox.
	Reset() error

	io.Closer
}

// Checker is an interface for a function that statelessly tries to evaluate a subtest from a submission
type Checker interface {
	Prepare(context.Context) (string, error)
	Cleanup(context.Context) error

	// RunChecker returns a comment and a number [0, 100] signifying the percentage of correctness of the subtest
	RunChecker(ctx context.Context, programOut, correctInput, correctOut io.Reader) (string, int)
}

type Runner interface {
	RunTask(context.Context, Task) error
	Close(context.Context) error
}

type Task interface {
	Execute(context.Context, Sandbox) error
}

type CompileRequest struct {
	ID          int
	CodeFiles   map[string][]byte
	HeaderFiles map[string][]byte
	Lang        string
}

type CompileResponse struct {
	Output  string
	Success bool
	Other   string
}

type ExecRequest struct {
	SubID       int
	SubtestID   int
	TestID      int
	Filename    string
	MemoryLimit int
	TimeLimit   float64
	Lang        string
}

type ExecResponse struct {
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

	TimeLimit     float64
	WallTimeLimit float64

	MaxProcs int

	InheritEnv   bool
	EnvToInherit []string
	EnvToSet     map[string]string

	Directories []Directory
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

// Limits stores the constraints that need to be respected by a submission
type Limits struct {
	// seconds
	TimeLimit float64
	// kilobytes
	MemoryLimit int
}

// GraderSubmission is an interface that provides a locked submission
type GraderSubmission interface {
	Submission() *kilonova.Submission
	Update(kilonova.SubmissionUpdate) error
	Close() error
}
