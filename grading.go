package kilonova

import "io"

// EVERYTHING IS A TODO HERE AND WILL CHANGE ACCORDING TO PLAN

type Sandbox interface {
	ReadFile(path string) (io.ReadSeekCloser, error)
	WriteFile(path string, w io.Writer) error
	DeleteFile(path string) error
	FileExists(path string) bool

	// if stdout == stderr, then it will act like exec.CombinedOutput()
	RunCommand(cmd []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (*RunStats, error)

	Close() error
}

type Sandboxer interface {
	NewSandbox() (Sandbox, error)
}

type Checker interface {
	RunChecker(programOut io.Reader, correctOut io.Reader, score int) (string, int, error)
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
