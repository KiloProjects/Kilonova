package box

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/KiloProjects/Kilonova/internal/languages"
)

var (
	releasePrefix = "https://github.com/KiloProjects/isolate/releases/download/v2.1/"
	configURL     = releasePrefix + "default.cf"
	configPath    = "/usr/local/etc/isolate"
	isolateURL    = releasePrefix + "isolate"
	isolatePath   string
)

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
	CgroupMem int32

	// Directories represents the list of mounted directories
	Directories []languages.Directory

	// Environment
	InheritEnv   bool
	EnvToInherit []string
	EnvToSet     map[string]string

	// Time limits (in seconds)
	TimeLimit      float64
	WallTimeLimit  float64
	ExtraTimeLimit float64

	InputFile  string
	OutputFile string
	ErrFile    string
	MetaFile   string

	// Memory limits (in kbytes)
	MemoryLimit int32
	StackSize   int32

	// Processes represents the maximum number of processes the program can create
	Processes int

	// Chdir is the directory (relative to the box root) to switch in
	Chdir string
}

// Box is the struct for the current box
type Box struct {
	path   string
	Config Config

	// Debug prints additional info if set
	Debug bool

	// the mutex makes sure we don't do anything stupid while we do other stuff
	mu sync.Mutex
}

// BuildRunFlags compiles all flags into an array
func (c *Config) BuildRunFlags() (res []string) {
	res = append(res, "--box-id="+strconv.Itoa(c.ID))

	if c.Cgroups {
		res = append(res, "--cg", "--cg-timing")
		if c.CgroupMem != 0 {
			res = append(res, "--cg-mem="+strconv.Itoa(int(c.CgroupMem)))
		}
	}
	for _, dir := range c.Directories {
		if dir.Removes {
			res = append(res, "--dir="+dir.In+"=")
			continue
		}
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
		memLim := approxMemory(c.MemoryLimit)
		res = append(res, "--mem="+strconv.Itoa(int(memLim)))
	}
	if c.StackSize != 0 {
		stackSize := approxMemory(c.StackSize)
		res = append(res, "--stack="+strconv.Itoa(int(stackSize)))
	}

	if c.Processes == 0 {
		res = append(res, "--processes")
	} else {
		res = append(res, "--processes="+strconv.Itoa(c.Processes))
	}

	if c.InputFile != "" {
		res = append(res, "--stdin="+c.InputFile)
	}
	if c.OutputFile != "" {
		res = append(res, "--stdout="+c.OutputFile)
	}
	if c.ErrFile != "" {
		res = append(res, "--stderr="+c.ErrFile)
	}
	if c.MetaFile != "" {
		res = append(res, "--meta="+c.MetaFile)
	}

	if c.Chdir != "" {
		res = append(res, "--chdir="+c.Chdir)
	}
	res = append(res, "--silent", "--run", "--")

	return
}

// WriteFile writes a file to the specified path inside the box
func (b *Box) WriteFile(fpath string, data []byte) error {
	return ioutil.WriteFile(b.GetFilePath(fpath), data, 0777)
}

// RemoveFile tries to remove a created file from inside the sandbox
func (b *Box) RemoveFile(fpath string) error {
	return os.Remove(b.GetFilePath(fpath))
}

// GetFile returns a file from inside the sandbox
func (b *Box) GetFile(fpath string) ([]byte, error) {
	return ioutil.ReadFile(b.GetFilePath(fpath))
}

// GetFilePath returns a path to the file location on disk of a box file
func (b *Box) GetFilePath(boxpath string) string {
	return path.Join(b.path, boxpath)
}

// CopyFromBox moves a file from the box to the outside world
// It will inherit all permissions
func (b *Box) CopyFromBox(boxpath string, rootpath string) error {
	return copyFile(b.GetFilePath(boxpath), rootpath)
}

// CopyInBox links a file to the box from the outside world
func (b *Box) CopyInBox(rootpath string, boxpath string) error {
	return copyFile(rootpath, b.GetFilePath(boxpath))
}

func copyFile(startpath string, endpath string) error {
	// just renaming doesn't work cross-disc (like moving to /tmp), so we create a new copy
	infile, err := os.Open(startpath)
	defer infile.Close()
	if err != nil {
		return err
	}

	stat, err := infile.Stat()
	if err != nil {
		return err
	}

	outfile, err := os.Create(endpath)
	defer outfile.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(outfile, infile)
	if err != nil {
		return err
	}

	return outfile.Chmod(stat.Mode())
}

// Cleanup is a convenience wrapper for cleanupBox
func (b *Box) Cleanup() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var params []string
	if b.Config.Cgroups {
		params = append(params, "--cg")
	}
	params = append(params, "--box-id="+strconv.Itoa(b.Config.ID), "--cleanup")
	return exec.Command(isolatePath, params...).Run()
}

// ExecCommand runs a command
func (b *Box) ExecCommand(command ...string) (string, string, error) {
	return b.ExecWithStdin("", command...)
}

// ExecCombinedOutput runs a command and returns the combined output
func (b *Box) ExecCombinedOutput(command ...string) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	params := append(b.Config.BuildRunFlags(), command...)
	cmd := exec.Command(isolatePath, params...)
	if b.Debug {
		fmt.Println("DEBUG:", cmd.String())
	}
	return cmd.CombinedOutput()
}

// ExecWithStdin runs a command with a specified stdin
// Returns the stdout, stderr and error (if anything bad happened)
func (b *Box) ExecWithStdin(stdin string, command ...string) (string, string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	params := append(b.Config.BuildRunFlags(), command...)
	cmd := exec.Command(isolatePath, params...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	if b.Debug {
		fmt.Println("DEBUG:", cmd.String())
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// New returns a new box instance from the specified ID
func New(config Config) (*Box, error) {
	ret, err := exec.Command(isolatePath, "--cg", fmt.Sprintf("--box-id=%d", config.ID), "--init").CombinedOutput()
	if strings.HasPrefix(string(ret), "Box already exists") {
		exec.Command(isolatePath, "--cg", fmt.Sprintf("--box-id=%d", config.ID), "--cleanup").Run()
		return New(config)
	}

	if strings.HasPrefix(string(ret), "Must be started as root") {
		if err := os.Chown(isolatePath, 0, 0); err != nil {
			fmt.Println("Couldn't chown root the isolate binary:", err)
			return nil, err
		}
		return New(config)
	}

	if err != nil {
		return nil, err
	}

	return &Box{path: strings.TrimSpace(string(ret)), Config: config}, nil
}

func downloadFile(url, path string, perm os.FileMode) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return file.Chmod(perm)
}

// Approximate to the nearest 128kb
func approxMemory(memory int32) int32 {
	rem := memory % 128
	if rem == 0 {
		return memory
	}
	return memory + 128 - rem
}

// Initialize should be called after reading the flags, but before manager.New
func Initialize(isolateBin string) error {
	isolatePath = isolateBin

	// Test right now if they exist
	if _, err := os.Stat(isolatePath); os.IsNotExist(err) {
		// download isolate
		fmt.Println("Downloading isolate binary")
		if err := downloadFile(isolateURL, isolatePath, 0744); err != nil {
			return err
		}
		fmt.Println("Isolate binary downloaded")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// download the config file
		fmt.Println("Downloading isolate config")
		if err := downloadFile(configURL, configPath, 0644); err != nil {
			return err
		}
		fmt.Println("Isolate config downloaded")
	}

	return nil
}
