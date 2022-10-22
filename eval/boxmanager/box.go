package boxmanager

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/davecgh/go-spew/spew"
)

var _ eval.Sandbox = &Box{}

// Env represents a variable-value pair for an environment variable
type Env struct {
	Var   string
	Value string
}

// Box is the struct for the current box
type Box struct {
	path  string
	boxID int

	// Debug prints additional info if set
	Debug bool

	// the mutex makes sure we don't do anything stupid while we do other stuff
	mu       sync.Mutex
	metaFile string
}

// buildRunFlags compiles all flags into an array
func (b *Box) buildRunFlags(c *eval.RunConfig) (res []string) {
	res = append(res, "--box-id="+strconv.Itoa(b.boxID))

	res = append(res, "--cg", "--cg-timing")
	/*if c.MemoryLimit != 0 {
		res = append(res, "--cg-mem="+strconv.Itoa(c.MemoryLimit))
	}*/
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

	if c.EnvToSet != nil {
		for key, val := range c.EnvToSet {
			res = append(res, "--env="+key+"="+val)
		}
	}

	if c.TimeLimit != 0 {
		res = append(res, "--time="+strconv.FormatFloat(c.TimeLimit, 'f', -1, 64))
	}
	if c.WallTimeLimit != 0 {
		res = append(res, "--wall-time="+strconv.FormatFloat(c.WallTimeLimit, 'f', -1, 64))
	}

	if c.MemoryLimit != 0 {
		res = append(res, "--mem="+strconv.Itoa(c.MemoryLimit))
	}

	if c.MaxProcs == 0 {
		res = append(res, "--processes")
	} else {
		res = append(res, "--processes="+strconv.Itoa(c.MaxProcs))
	}

	if c.InputPath != "" {
		res = append(res, "--stdin="+c.InputPath)
	}
	if c.OutputPath != "" {
		res = append(res, "--stdout="+c.OutputPath)
	}
	if b.metaFile != "" {
		res = append(res, "--meta="+b.metaFile)
	}

	res = append(res, "--silent", "--run", "--")

	return
}

// WriteFile writes a file to the specified path inside the box
func (b *Box) WriteFile(fpath string, r io.Reader, mode fs.FileMode) error {
	return writeReader(b.getFilePath(fpath), r, mode)
}

func (b *Box) ReadFile(fpath string) (io.ReadSeekCloser, error) {
	return os.Open(b.getFilePath(fpath))
}

func (b *Box) GetID() int {
	return b.boxID
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

func (b *Box) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	var params []string
	params = append(params, "--cg")
	params = append(params, "--box-id="+strconv.Itoa(b.boxID), "--cleanup")
	return exec.Command(config.Eval.IsolatePath, params...).Run()
}

func (b *Box) RunCommand(ctx context.Context, command []string, conf *eval.RunConfig) (*eval.RunStats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	metaFile := path.Join(os.TempDir(), "kn-"+kilonova.RandomString(6))
	b.metaFile = metaFile

	params := append(b.buildRunFlags(conf), command...)
	cmd := exec.CommandContext(ctx, config.Eval.IsolatePath, params...)

	b.metaFile = ""

	if b.Debug {
		// fmt.Println("DEBUG:", cmd.String())
	}

	if conf != nil {
		cmd.Stdin = conf.Stdin
		cmd.Stdout = conf.Stdout
		cmd.Stderr = conf.Stderr
	}
	err := cmd.Run()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		spew.Dump(err)
		return nil, err
	}

	// read Meta File
	f, err := os.Open(metaFile)
	if err != nil {
		fmt.Println("Couldn't open meta file, wtf", err)
		return nil, nil
	}
	defer f.Close()
	defer os.Remove(metaFile)
	return parseMetaFile(f), nil
}

func (b *Box) Reset() error {
	if err := b.Close(); err != nil {
		return err
	}
	box, err := newBox(b.boxID)
	if err != nil {
		return err
	}
	b = box
	return nil
}

// newBox returns a new box instance from the specified ID
func newBox(id int) (*Box, error) {
	ret, err := exec.Command(config.Eval.IsolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--init").CombinedOutput()
	if strings.HasPrefix(string(ret), "Box already exists") {
		exec.Command(config.Eval.IsolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--cleanup").Run()
		return newBox(id)
	}

	if strings.HasPrefix(string(ret), "Must be started as root") {
		if err := os.Chown(config.Eval.IsolatePath, 0, 0); err != nil {
			fmt.Println("Couldn't chown root the isolate binary:", err)
			return nil, err
		}
		return newBox(id)
	}

	if err != nil {
		return nil, err
	}

	return &Box{path: strings.TrimSpace(string(ret)), boxID: id}, nil
}

func CheckCanRun() bool {
	box, err := newBox(0)
	if err != nil {
		log.Println(err)
		return false
	}
	if err := box.Close(); err != nil {
		log.Println(err)
		return false
	}
	return true
}

func writeReader(path string, r io.Reader, perms fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

// parseMetaFile parses a specified meta file
func parseMetaFile(r io.Reader) *eval.RunStats {
	if r == nil {
		return nil
	}
	var file = new(eval.RunStats)

	s := bufio.NewScanner(r)

	for s.Scan() {
		if !strings.Contains(s.Text(), ":") {
			continue
		}
		l := strings.SplitN(s.Text(), ":", 2)
		switch l[0] {
		case "cg-mem":
			file.Memory, _ = strconv.Atoi(l[1])
		case "exitcode":
			file.ExitCode, _ = strconv.Atoi(l[1])
		case "exitsig":
			file.ExitSignal, _ = strconv.Atoi(l[1])
		case "killed":
			file.Killed = true
		case "message":
			file.Message = l[1]
		case "status":
			file.Status = l[1]
		case "time":
			file.Time, _ = strconv.ParseFloat(l[1], 32)
		case "time-wall":
			file.WallTime, _ = strconv.ParseFloat(l[1], 32)
		default:
			continue
		}
	}

	return file
}
