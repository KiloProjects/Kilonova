package eval

import (
	"context"
	"io"
	"io/fs"

	"go.uber.org/zap"
)

type Sandbox interface {
	// ReadFile reads contents of path from sandbox and pipes them to the given writer
	ReadFile(path string, w io.Writer) error
	WriteFile(path string, r io.Reader, mode fs.FileMode) error
	// RemoveFile(path string) error
	FileExists(path string) bool

	GetID() int
	MemoryQuota() int64

	// if stdout == stderr, then it will act like exec.CombinedOutput()
	RunCommand(ctx context.Context, cmd []string, conf *RunConfig) (*RunStats, error)

	io.Closer
}

type BoxScheduler interface {
	SubRunner(ctx context.Context, numConc int64) (BoxScheduler, error)
	NumConcurrent() int64
	GetBox(ctx context.Context, memQuota int64) (Sandbox, error)
	ReleaseBox(Sandbox)
	Close(context.Context) error
}

type Task[Req, Resp any] func(context.Context, Sandbox, *Req) (*Resp, error)

func (t Task[Req, Resp]) Run(ctx context.Context, mgr BoxScheduler, memQuota int64, r *Req) (*Resp, error) {
	box, err := mgr.GetBox(ctx, memQuota)
	if err != nil {
		zap.S().Info(err)
		return nil, err
	}
	defer mgr.ReleaseBox(box)
	return t(ctx, box, r)
}

type RunConfig struct {
	StderrToStdout bool

	InputPath  string
	OutputPath string
	StderrPath string

	MemoryLimit int

	TimeLimit     float64
	WallTimeLimit float64

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

	Time float64 `json:"time"`

	InternalMessage string `json:"internal_msg"`
	// WallTime float64 `json:"wall_time"`
}
