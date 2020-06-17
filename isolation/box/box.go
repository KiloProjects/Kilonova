package box

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

// Directory represents a directory rule
type Directory struct {
	In   string
	Out  string
	Opts string
}

// Env represents a variable-value pair for an environment variable
type Env struct {
	Var   string
	Value string
}

// Config is the struct that controls the sandbox
type Config struct {
	// Box ID
	ID int

	// Mark if Cgroups should be enabled
	Cgroups bool
	// Maximum Cgroup memory (in kbytes)
	CgroupMem int

	// Directories represents the list of mounted directories
	Directories []Directory

	// Environment
	InheritEnv   bool
	EnvToInherit []string
	EnvToSet     map[string]string

	// Time limits (in seconds)
	TimeLimit      float64
	WallTimeLimit  float64
	ExtraTimeLimit float64

	// Memory limits (in kbytes)
	MemoryLimit int
	StackSize   int

	// Processes represents the maximum number of processes the program can create
	Processes int

	// Chdir is the directory (relative to the box root) to switch in
	Chdir string
}

// Box is the struct for the current box
type Box struct {
	path   string
	Config Config
}

// BuildRunFlags compiles all flags into an array
func (c *Config) BuildRunFlags() (res []string) {
	res = append(res, "--box-id="+strconv.Itoa(c.ID))

	if c.Cgroups {
		res = append(res, "--cg", "--cg-timing")
		if c.CgroupMem != 0 {
			res = append(res, "--cg-mem="+strconv.Itoa(c.CgroupMem))
		}
	}

	for _, dir := range c.Directories {
		toAdd := "--dir="
		toAdd += dir.In + "="
		if dir.Out == "" {
			toAdd += dir.In
		} else {
			toAdd += dir.Out
		}
		if dir.Opts != "" {
			toAdd += ":" + dir.Opts
		}
		res = append(res, toAdd)
	}

	if c.InheritEnv {
		res = append(res, "--full-env")
	}
	for _, env := range c.EnvToInherit {
		res = append(res, "--env="+env)
	}
	for key, val := range c.EnvToSet {
		res = append(res, "--env="+key+"="+val)
	}

	if c.TimeLimit != 0 {
		res = append(res, "--time="+strconv.FormatFloat(c.TimeLimit, 'f', -1, 64))
	}
	if c.WallTimeLimit != 0 {
		res = append(res, "--wall-time="+strconv.FormatFloat(c.WallTimeLimit, 'f', -1, 64))
	}
	if c.ExtraTimeLimit != 0 {
		res = append(res, "--extra-time="+strconv.FormatFloat(c.ExtraTimeLimit, 'f', -1, 64))
	}

	if c.MemoryLimit != 0 {
		res = append(res, "--mem="+strconv.Itoa(c.MemoryLimit))
	}
	if c.StackSize != 0 {
		res = append(res, "--stack="+strconv.Itoa(c.StackSize))
	}

	if c.Processes == 0 {
		res = append(res, "--processes")
	} else {
		res = append(res, "--processes="+strconv.Itoa(c.Processes))
	}

	if c.Chdir != "" {
		res = append(res, "--chdir="+c.Chdir)
	}
	res = append(res, "--run", "--")
	return
}

// WriteFile writes a file to the specified filepath *inside the box*
func (b *Box) WriteFile(filepath, data string) error {
	return ioutil.WriteFile(path.Join(b.path, filepath), []byte(data), 0777)
}

// Cleanup is a convenience wrapper for cleanupBox
func (b *Box) Cleanup() error {
	var params []string
	if b.Config.Cgroups {
		params = append(params, "--cg")
	}
	params = append(params, "--box-id="+strconv.Itoa(b.Config.ID), "--cleanup")
	return exec.Command("isolate", params...).Run()
}

// ExecCommand runs a command
func (b *Box) ExecCommand(command ...string) (string, error) {
	return b.ExecWithStdin("", command...)
}

// ExecWithStdin runs the command with the specified stdin
func (b *Box) ExecWithStdin(stdin string, command ...string) (string, error) {
	params := append(b.Config.BuildRunFlags(), command...)
	cmd := exec.Command("isolate", params...)
	cmd.Stdin = strings.NewReader(stdin)
	fmt.Println(cmd.String())
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// NewBox returns a new box instance from the specified ID
func NewBox(config Config) *Box {
	ret, _ := exec.Command("isolate", "--cg", fmt.Sprintf("--box-id=%d", config.ID), "--init").CombinedOutput()
	return &Box{path: strings.TrimSpace(string(ret)), Config: config}
}
