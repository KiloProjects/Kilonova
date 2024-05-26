package eval

import (
	"context"
	"io"
	"io/fs"
)

type Bucket interface {
	// It should also implement slog.LogValuer for pretty printing in debug output
	// But it's not really necessary

	Reader(name string) (io.ReadCloser, error)
	Stat(name string) (fs.FileInfo, error)
	WriteFile(name string, r io.Reader, mode fs.FileMode) error
}

type Sandbox interface {
	// ReadFile reads contents of path from sandbox and pipes them to the given writer
	ReadFile(path string, w io.Writer) error
	// SaveFile reads contents of path from sandbox and saves them in the given bucket by calling WriteFile
	SaveFile(path string, bucket Bucket, filename string, mode fs.FileMode) error
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
	RunBox2(ctx context.Context, req *Box2Request, memQuota int64) (*Box2Response, error)
	Close(context.Context) error

	LanguageVersions(ctx context.Context) map[string]string
}

type RunConfig struct {
	StderrToStdout bool

	InputPath string
	// If OutputPath or StderrPath are empty strings, they should default
	// to "/dev/null" for security reasons.
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
