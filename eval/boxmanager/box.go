package boxmanager

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

const (
	runErrRetries = 3
	runErrTimeout = 200 * time.Millisecond
)

var _ eval.Sandbox = &Box{}

// Env represents a variable-value pair for an environment variable
type Env struct {
	Var   string
	Value string
}

// Box is the struct for the current box
type Box struct {
	// the mutex makes sure we don't do anything stupid while we do other stuff
	mu    sync.Mutex
	path  string
	boxID int

	memoryQuota int64

	metaFile string

	logger *zap.SugaredLogger
}

// buildRunFlags compiles all flags into an array
func (b *Box) buildRunFlags(c *eval.RunConfig) (res []string) {
	res = append(res, "--box-id="+strconv.Itoa(b.boxID))

	res = append(res, "--cg", "--cg-timing")
	for _, dir := range c.Directories {
		if dir.Removes {
			res = append(res, "--dir="+dir.In+"=")
			continue
		}
		toAdd := "--dir="
		toAdd += dir.In
		if dir.Out == "" {
			if !dir.Verbatim {
				toAdd += "=" + dir.In
			}
		} else {
			toAdd += "=" + dir.Out
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
		if b.memoryQuota > 0 && int64(c.MemoryLimit) > b.memoryQuota {
			zap.S().Info("Memory limit supplied exceeds quota")
			c.MemoryLimit = int(b.memoryQuota)
		}
		res = append(res, "--cg-mem="+strconv.Itoa(c.MemoryLimit))
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

	if c.StderrPath != "" {
		res = append(res, "--stderr="+c.StderrPath)
	} else if c.StderrToStdout {
		res = append(res, "--stderr-to-stdout")
	}

	if b.metaFile != "" {
		res = append(res, "--meta="+b.metaFile)
	}

	res = append(res, "--silent", "--run", "--")

	return
}

// WriteFile writes a file to the specified path inside the box
func (b *Box) WriteFile(fpath string, r io.Reader, mode fs.FileMode) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return writeReader(b.getFilePath(fpath), r, mode)
}

func (b *Box) ReadFile(fpath string, w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	f, err := os.Open(b.getFilePath(fpath))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func (b *Box) ReadDir(fpath string) ([]string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := b.getFilePath(fpath)
	d, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, file := range d {
		out = append(out, file.Name())
	}
	return out, nil
}

func (b *Box) GetID() int {
	return b.boxID
}

func (b *Box) MemoryQuota() int64 {
	return b.memoryQuota
}

// FileExists returns if a file exists or not
func (b *Box) FileExists(fpath string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, err := os.Stat(b.getFilePath(fpath))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
		if strings.Contains(err.Error(), "permission denied") { // Try to diagnose the race condition if it happens
			zap.S().Warnf("File stat (%q) returned the race condition error: %s", fpath, err)
			zap.S().Info("Printing stat results of parents...")
			x := strings.Split(b.getFilePath(fpath), "/")
			x[0] = "/" + x[0]
			for i := range x {
				pp := path.Join(x[:i]...)
				if pp == "" {
					continue
				}
				if _, err := os.Stat(pp); err == nil {
					zap.S().Infof("%q is ok", pp)
				} else {
					zap.S().Infof("%q churns out error: %s", pp, err)
				}
				if strings.HasPrefix(pp, "/var/local/lib/isolate/") {
					zap.S().Infof("Running ls on %q...", pp)
					out, err := exec.Command("/usr/bin/ls", "-la", "--author", pp).CombinedOutput()
					fmt.Println(string(out))
					spew.Dump(err)
				}
			}
			zap.S().Info("Checking again")
			_, err = os.Stat(b.getFilePath(fpath))
			if err != nil {
				zap.S().Infof("Still errors: %s", err)
			} else {
				zap.S().Info("It... works?")
				return true
			}
			return false
		}
		zap.S().Warnf("File stat (%q) returned weird error: %s", fpath, err)
		return false
	}
	return true
}

// getFilePath returns a path to the file location on disk of a box file
func (b *Box) getFilePath(boxpath string) string {
	return path.Join(b.path, boxpath)
}

func (b *Box) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return exec.Command(config.Eval.IsolatePath, "--cg", "--box-id="+strconv.Itoa(b.boxID), "--cleanup").Run()
}

func (b *Box) runCommand(ctx context.Context, params []string, metaFile string) (*eval.RunStats, error) {
	cmd := exec.CommandContext(ctx, config.Eval.IsolatePath, params...)

	err := cmd.Run()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		spew.Dump(err)
		return nil, err
	}

	// read Meta File
	f, err := os.Open(metaFile)
	if err != nil {
		zap.S().Warn("Couldn't open meta file, wtf: ", err)
		return nil, nil
	}
	defer f.Close()
	defer os.Remove(metaFile)
	return parseMetaFile(f), nil
}

func (b *Box) RunCommand(ctx context.Context, command []string, conf *eval.RunConfig) (*eval.RunStats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var meta *eval.RunStats
	var err error

	for i := 1; i <= runErrRetries; i++ {
		metaFile := path.Join(os.TempDir(), "kn-"+kilonova.RandomString(12))
		b.metaFile = metaFile
		meta, err = b.runCommand(ctx, append(b.buildRunFlags(conf), command...), metaFile)
		b.metaFile = ""
		if err == nil && meta != nil && meta.Status != "XX" {
			return meta, err
		}

		zap.S().Warnf("Run error in box %d, retrying (%d/%d). Check grader.log for more details", b.boxID, i, runErrRetries)
		b.logger.Warnf("Run error in box %d, retrying (%d/%d): '%#v' %s", b.boxID, i, runErrRetries, err, spew.Sdump(meta))
		time.Sleep(runErrTimeout)
	}

	return meta, nil
}

// newBox returns a new box instance from the specified ID
func newBox(id int, memQuota int64, logger *zap.SugaredLogger) (*Box, error) {
	ret, err := exec.Command(config.Eval.IsolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--init").CombinedOutput()
	if strings.HasPrefix(string(ret), "Box already exists") {
		zap.S().Info("Box reset: ", id)
		if out, err := exec.Command(config.Eval.IsolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--cleanup").CombinedOutput(); err != nil {
			zap.S().Warn(err, string(out))
		}
		return newBox(id, memQuota, logger)
	}

	if strings.HasPrefix(string(ret), "Must be started as root") {
		if err := os.Chown(config.Eval.IsolatePath, 0, 0); err != nil {
			fmt.Println("Couldn't chown root the isolate binary:", err)
			return nil, err
		}
		return newBox(id, memQuota, logger)
	}

	if err != nil {
		return nil, err
	}

	return &Box{path: strings.TrimSpace(string(ret)), boxID: id, memoryQuota: memQuota, logger: logger}, nil
}

func CheckCanRun() bool {
	box, err := newBox(0, 0, zap.S())
	if err != nil {
		zap.S().Warn(err)
		return false
	}
	if err := box.Close(); err != nil {
		zap.S().Warn(err)
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
		case "max-rss", "csw-voluntary", "csw-forced", "cg-enabled", "cg-oom-killed":
			continue
		default:
			zap.S().Infof("Unknown isolate stat: %q (value: %v)", l[0], l[1])
			continue
		}
	}

	return file
}
