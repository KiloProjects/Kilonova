package box

import (
	"bufio"
	"bytes"
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

var _ eval.Sandbox = &IsolateBox{}

// IsolateBox is the struct for the current box
type IsolateBox struct {
	// the mutex makes sure we don't do anything stupid while we do other stuff
	mu    sync.Mutex
	path  string
	boxID int

	memoryQuota int64

	metaFile string

	logger *zap.SugaredLogger
}

var CGTiming = config.GenFlag[bool]("feature.grader.use_cg_timing", false, "Use --cg-timing flag in grader. Should not be necessary.")

// buildRunFlags compiles all flags into an array
func (b *IsolateBox) buildRunFlags(c *eval.RunConfig) (res []string) {
	res = append(res, "--box-id="+strconv.Itoa(b.boxID))

	res = append(res, "--cg", "--processes")
	if CGTiming.Value() {
		res = append(res, "--cg-timing")
	}
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

	if c.InputPath == "" {
		c.InputPath = "/dev/null"
	}
	if c.OutputPath == "" {
		c.OutputPath = "/dev/null"
	}
	res = append(res, "--stdin="+c.InputPath)
	res = append(res, "--stdout="+c.OutputPath)

	if c.StderrToStdout {
		res = append(res, "--stderr-to-stdout")
	} else {
		if c.StderrPath == "" {
			c.StderrPath = "/dev/null"
		}
		res = append(res, "--stderr="+c.StderrPath)
	}

	if b.metaFile != "" {
		res = append(res, "--meta="+b.metaFile)
	}

	res = append(res, "--silent", "--run", "--")

	return
}

// WriteFile writes an eval file to the specified path inside the box
func (b *IsolateBox) WriteFile(fpath string, r io.Reader, mode fs.FileMode) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return writeFile(b.getFilePath(fpath), r, mode)
}

func (b *IsolateBox) ReadFile(fpath string, w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return readFile(b.getFilePath(fpath), w)
}

func (b *IsolateBox) GetID() int {
	return b.boxID
}

func (b *IsolateBox) MemoryQuota() int64 {
	return b.memoryQuota
}

// FileExists returns if a file exists or not
func (b *IsolateBox) FileExists(fpath string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return checkFile(b.getFilePath(fpath))
}

// getFilePath returns a path to the file location on disk of a box file
func (b *IsolateBox) getFilePath(boxpath string) string {
	return path.Join(b.path, boxpath)
}

func (b *IsolateBox) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return exec.Command(config.Eval.IsolatePath, "--cg", "--box-id="+strconv.Itoa(b.boxID), "--cleanup").Run()
}

func (b *IsolateBox) runCommand(ctx context.Context, params []string, metaFile string) (*eval.RunStats, error) {
	var isolateOut bytes.Buffer
	cmd := exec.CommandContext(ctx, config.Eval.IsolatePath, params...)
	cmd.Stdout = &isolateOut
	cmd.Stderr = &isolateOut
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
	return parseMetaFile(f, isolateOut), nil
}

func dumpFileListing(w io.Writer, p string, showPath string, indent string, rec bool) {
	entries, err := os.ReadDir(p)
	if err != nil {
		fmt.Fprintf(w, "%sCould not read `%s`: %#v", indent, showPath, err)
	} else {
		fmt.Fprintf(w, "%s- `%s` contents:\n", indent, showPath)
		for _, entry := range entries {
			mode := "???"
			info, err := entry.Info()
			if err != nil {
				mode = "ERR:" + err.Error()
			} else {
				mode = info.Mode().String()
			}

			fmt.Fprintf(w, "%s\t`%s` (mode: %s) size: %d\n", indent, info.Name(), mode, info.Size())
			if info.IsDir() && rec {
				dumpFileListing(w, path.Join(p, info.Name()), path.Join(showPath, info.Name()), indent+"\t", rec)
			}

		}
	}
}

func (b *IsolateBox) RunCommand(ctx context.Context, command []string, conf *eval.RunConfig) (*eval.RunStats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var meta *eval.RunStats
	var err error

	if strings.HasPrefix(command[0], "/box") {
		p := b.getFilePath(command[0])
		if _, err := os.Stat(p); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				zap.S().Warnf("Executable does not exist in sandbox and will probably error in box %d", b.boxID)
			} else {
				zap.S().Warn(err)
			}
		}
	}

	for i := 1; i <= runErrRetries; i++ {
		metaFile := path.Join(os.TempDir(), "kn-"+kilonova.RandomString(12))
		b.metaFile = metaFile
		meta, err = b.runCommand(ctx, append(b.buildRunFlags(conf), command...), metaFile)
		b.metaFile = ""
		if err == nil && meta != nil && meta.Status != "XX" {
			if meta.ExitCode == 127 {
				if strings.Contains(meta.InternalMessage, "execve") { // It's text file busy, most likely...
					// Not yet marked as a stable solution
					// if i > 1 {
					// 	// Only warn if it comes to the second attempt. First error is often enough in prod
					zap.S().Warnf("Text file busy error in box %d, retrying (%d/%d). Check grader.log for more details", b.boxID, i, runErrRetries)
					// }
					b.logger.Warnf("Text file busy error in box %d, retrying (%d/%d): %s", b.boxID, i, runErrRetries, spew.Sdump(meta))
					time.Sleep(runErrTimeout)
					continue
				}
				zap.S().Warnf("Exit code 127 in box %d (not execve!). Check grader.log for more details", b.boxID)
				var s strings.Builder
				fmt.Fprintf(&s, "Exit code 127 in box %d\n", b.boxID)
				spew.Fdump(&s, conf)
				fmt.Fprintf(&s, "Command: %#+v\n", command)
				fmt.Fprintf(&s, "Isolate out: %q\n", meta.InternalMessage)
				dumpFileListing(&s, b.getFilePath("/"), "/", "", true)
				b.logger.Warn(s.String())
			}
			return meta, err
		}

		if i > 1 {
			// Only warn if it comes to the second attempt. First error is often enough in prod
			zap.S().Warnf("Run error in box %d, retrying (%d/%d). Check grader.log for more details", b.boxID, i, runErrRetries)
		}
		b.logger.Warnf("Run error in box %d, retrying (%d/%d): '%#v' %s", b.boxID, i, runErrRetries, err, spew.Sdump(meta))
		time.Sleep(runErrTimeout)
	}

	return meta, nil
}

// New returns a new box instance from the specified ID
func New(id int, memQuota int64, logger *zap.SugaredLogger) (eval.Sandbox, error) {
	ret, err := exec.Command(config.Eval.IsolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--init").CombinedOutput()
	if strings.HasPrefix(string(ret), "Box already exists") {
		zap.S().Info("Box reset: ", id)
		if out, err := exec.Command(config.Eval.IsolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--cleanup").CombinedOutput(); err != nil {
			zap.S().Warn(err, string(out))
		}
		return New(id, memQuota, logger)
	}

	if strings.Contains(string(ret), "incompatible control group mode") { // Created without --cg
		zap.S().Info("Box reset: ", id)
		if out, err := exec.Command(config.Eval.IsolatePath, fmt.Sprintf("--box-id=%d", id), "--cleanup").CombinedOutput(); err != nil {
			zap.S().Warn(err, string(out))
		}
		return New(id, memQuota, logger)
	}

	if strings.HasPrefix(string(ret), "Must be started as root") {
		if err := os.Chown(config.Eval.IsolatePath, 0, 0); err != nil {
			fmt.Println("Couldn't chown root the isolate binary:", err)
			return nil, err
		}
		return New(id, memQuota, logger)
	}

	if err != nil {
		return nil, err
	}

	return &IsolateBox{path: strings.TrimSpace(string(ret)), boxID: id, memoryQuota: memQuota, logger: logger}, nil
}

func IsolateVersion() string {
	ret, err := exec.Command(config.Eval.IsolatePath, "--version").CombinedOutput()
	if err != nil {
		return "precompiled?"
	}
	line, _, _ := bytes.Cut(ret, []byte{'\n'})
	return strings.TrimPrefix(string(line), "The process isolator ")
}

// parseMetaFile parses a specified meta file
func parseMetaFile(r io.Reader, out bytes.Buffer) *eval.RunStats {
	if r == nil {
		return nil
	}
	var file = new(eval.RunStats)

	file.InternalMessage = out.String()

	s := bufio.NewScanner(r)

	for s.Scan() {
		key, val, found := strings.Cut(s.Text(), ":")
		if !found {
			continue
		}
		switch key {
		case "cg-mem":
			file.Memory, _ = strconv.Atoi(val)
		case "exitcode":
			file.ExitCode, _ = strconv.Atoi(val)
		case "exitsig":
			file.ExitSignal, _ = strconv.Atoi(val)
		case "killed":
			file.Killed = true
		case "message":
			file.Message = val
		case "status":
			file.Status = val
		case "time":
			file.Time, _ = strconv.ParseFloat(val, 64)
		case "time-wall":
			// file.WallTime, _ = strconv.ParseFloat(val, 32)
			continue
		case "max-rss", "csw-voluntary", "csw-forced", "cg-enabled", "cg-oom-killed":
			continue
		default:
			zap.S().Infof("Unknown isolate stat: %q (value: %v)", key, val)
			continue
		}
	}

	return file
}
