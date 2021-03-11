package kilonova

import (
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/KiloProjects/kilonova/internal/config"
)

// EVERYTHING IS A TODO HERE AND WILL CHANGE ACCORDING TO PLAN

type Sandbox interface {
	ReadFile(path string) (io.ReadSeekCloser, error)
	WriteFile(path string, r io.Reader, mode fs.FileMode) error
	RemoveFile(path string) error
	FileExists(path string) bool

	GetID() int

	// if stdout == stderr, then it will act like exec.CombinedOutput()
	RunCommand(ctx context.Context, cmd []string, conf *RunConfig) (*RunStats, error)

	Close() error
}

// Checker is an interface for a function that statelessly tries to evaluate a subtest from a submission
type Checker interface {
	Prepare(context.Context) error
	Cleanup(context.Context) error
	RunChecker(ctx context.Context, programOut, correctOut io.Reader, maxScore int) (string, int)
}

type Runner interface {
	RunJob(context.Context, Job) error

	Compile(context.Context, *CompileRequest) (*CompileResponse, error)
	Execute(context.Context, *ExecRequest) (*ExecResponse, error)
	Clean(ctx context.Context, subid int) error
	Close(context.Context) error

	GetSandbox(context.Context) (Sandbox, error)
	ReleaseSandbox(Sandbox)
}

type Job interface {
	Execute(context.Context, Sandbox) error
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

	TimeLimit     float64
	WallTimeLimit float64

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

func CopyFromBox(b Sandbox, p string, w io.Writer) error {
	f, err := b.ReadFile(p)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func CopyInBox(b Sandbox, p1 string, p2 string) error {
	file, err := os.Open(p1)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	return b.WriteFile(p2, file, stat.Mode())
}

func LangToRunConf(language config.Language) *RunConfig {
	var runConf RunConfig
	runConf.EnvToSet = make(map[string]string)

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !language.IsCompiled {
		runConf.Directories = append(runConf.Directories, language.Mounts...)
	}

	for key, val := range language.CommonEnv {
		runConf.EnvToSet[key] = val
	}
	for key, val := range language.RunEnv {
		runConf.EnvToSet[key] = val
	}

	return &runConf
}
