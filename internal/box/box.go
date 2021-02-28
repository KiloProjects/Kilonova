package box

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
)

var (
	releasePrefix = "https://github.com/KiloProjects/isolate/releases/latest/download/"
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
	Directories []config.Directory

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
func (b *Box) WriteFile(fpath string, r io.Reader) error {
	return writeReader(b.getFilePath(fpath), r, 0777)
}

// FileExists returns if a file exists or not
func (b *Box) FileExists(fpath string) bool {
	_, err := os.Stat(b.getFilePath(fpath))
	if err != nil {
		// TODO: Only fs.ErrNotExist should happen, make sure it is that way
		return false
	}
	return true
}

// RemoveFile tries to remove a created file from inside the sandbox
func (b *Box) RemoveFile(fpath string) error {
	return os.Remove(b.getFilePath(fpath))
}

// getFilePath returns a path to the file location on disk of a box file
func (b *Box) getFilePath(boxpath string) string {
	return path.Join(b.path, boxpath)
}

// CopyFromBox copies a file from the box to the writer
// It will inherit all permissions
func (b *Box) CopyFromBox(boxpath string, w io.Writer) error {
	f, err := os.Open(b.getFilePath(boxpath))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

// CopyInBox copies a file to the box from the outside world
// It will inherit all permissions
func (b *Box) CopyInBox(rootpath, boxpath string) error {
	return copyFile(rootpath, b.getFilePath(boxpath))
}

func copyFile(startpath, endpath string) error {
	// just renaming doesn't work cross-disc (like moving to /tmp), so we create a new copy
	infile, err := os.Open(startpath)
	if err != nil {
		return err
	}
	defer infile.Close()

	stat, err := infile.Stat()
	if err != nil {
		return err
	}

	return writeReader(endpath, infile, stat.Mode())
}

func (b *Box) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var params []string
	if b.Config.Cgroups {
		params = append(params, "--cg")
	}
	params = append(params, "--box-id="+strconv.Itoa(b.Config.ID), "--cleanup")
	return exec.Command(isolatePath, params...).Run()
}

func (b *Box) RunCommand(command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (*kilonova.RunStats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	params := append(b.Config.BuildRunFlags(), command...)
	cmd := exec.Command(isolatePath, params...)

	if b.Debug {
		fmt.Println("DEBUG:", cmd.String())
	}

	metaFile := path.Join(os.TempDir(), "kn-"+kilonova.RandomString(6))
	b.Config.MetaFile = metaFile

	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// read Meta File
	f, err := os.Open(metaFile)
	if err != nil {
		return nil, nil
	}
	defer f.Close()
	return ParseMetaFile(f), nil
}

// ExecCombinedOutput runs a command and returns the combined output
func (b *Box) ExecCombinedOutput(command ...string) ([]byte, error) {
	var output bytes.Buffer
	_, err := b.RunCommand(command, nil, &output, &output)
	return output.Bytes(), err
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

func writeReader(path string, r io.Reader, perms fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	_, err = f.ReadFrom(r)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
