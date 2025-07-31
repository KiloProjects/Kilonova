package box

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/davecgh/go-spew/spew"
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

	logger *slog.Logger
}

// buildRunFlags compiles all flags into an array
func (b *IsolateBox) buildRunFlags(ctx context.Context, c *eval.RunConfig, metaFilePath string) (res []string) {
	res = append(res, "--box-id="+strconv.Itoa(b.boxID))

	res = append(res, "--cg", "--processes")
	//res = append(res, "--cg-timing")
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
			slog.InfoContext(ctx, "Memory limit supplied exceeds quota", slog.Int64("quota", b.memoryQuota), slog.Int("target_limit", c.MemoryLimit))
			c.MemoryLimit = int(b.memoryQuota)
		}
		res = append(res, "--cg-mem="+strconv.Itoa(c.MemoryLimit))
	} else if b.memoryQuota > 0 {
		// Still include a memory limit if quota is defined.
		// Just a sanity check to ensure resources aren't exhausted.
		res = append(res, "--cg-mem="+strconv.FormatInt(b.memoryQuota, 10))
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

	if len(metaFilePath) > 0 {
		res = append(res, "--meta="+metaFilePath)
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

func (b *IsolateBox) SaveFile(fpath string, bucket eval.Bucket, filename string, mode fs.FileMode) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return saveFile(b.getFilePath(fpath), bucket, filename, mode)
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
	return exec.Command(isolatePath, "--cg", "--box-id="+strconv.Itoa(b.boxID), "--cleanup").Run()
}

func (b *IsolateBox) runCommand(ctx context.Context, params []string, metaFile *os.File) (*eval.RunStats, error) {
	var isolateOut bytes.Buffer
	cmd := exec.CommandContext(ctx, isolatePath, params...)
	cmd.Stdout = &isolateOut
	cmd.Stderr = &isolateOut
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	err := cmd.Run()
	var exitError *exec.ExitError
	if err != nil && !errors.As(err, &exitError) && !errors.Is(err, context.Canceled) {
		spew.Dump(err)
		return nil, err
	}

	// read Meta File
	defer metaFile.Close()
	defer os.Remove(metaFile.Name())
	return parseMetaFile(ctx, metaFile, isolateOut), nil
}

func dumpFileListing(w io.Writer, p string, showPath string, indent string, rec bool) {
	entries, err := os.ReadDir(p)
	if err != nil {
		fmt.Fprintf(w, "%sCould not read `%s`: %#v", indent, showPath, err)
	} else {
		fmt.Fprintf(w, "%s- `%s` contents:\n", indent, showPath)
		for _, entry := range entries {
			var mode string
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

	if strings.HasPrefix(command[0], "/box") {
		p := b.getFilePath(command[0])
		if _, err := os.Stat(p); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				slog.WarnContext(ctx, "Executable does not exist in sandbox and will probably error", slog.Int("box_id", b.boxID))
			} else {
				slog.WarnContext(ctx, "Could not find file to run in sandbox", slog.Any("err", err))
			}
		}
	}

	if conf.MemoryLimit > 0 {
		for i := range command {
			if strings.Contains(command[i], eval.MemoryReplace) {
				command[i] = strings.ReplaceAll(command[i], eval.MemoryReplace, strconv.Itoa(conf.MemoryLimit))
			}
		}
	}

	for i := 1; i <= runErrRetries; i++ {
		metaFile, err := os.CreateTemp("", "kn-meta-*")
		if err != nil {
			slog.WarnContext(ctx, "Couldn't create meta file", slog.Any("err", err))
			continue
		}
		meta, err = b.runCommand(ctx, append(b.buildRunFlags(ctx, conf, metaFile.Name()), command...), metaFile)
		if err == nil && meta != nil && meta.Status != "XX" {
			if meta.ExitCode == 127 {
				if strings.Contains(meta.InternalMessage, "execve") { // It's text file busy, most likely...
					// Not yet marked as a stable solution
					// if i > 1 {
					// 	// Only warn if it comes to the second attempt. First error is often enough in prod
					slog.WarnContext(ctx, "Text file busy error in sandbox, will retry. Check grader.log for more details", slog.Int("box_id", b.boxID), slog.Int("attempt", i), slog.Int("max_retries", runErrRetries))
					// }
					b.logger.WarnContext(ctx, "Text file busy error, retrying", slog.Int("box_id", b.boxID), slog.Int("attempt", i), slog.Int("max_retries", runErrRetries), slog.Any("metadata", meta))
					time.Sleep(runErrTimeout)
					continue
				}
				slog.WarnContext(ctx, "Exit code 127 in sandbox (not execve). Check grader.log for more details", slog.Int("box_id", b.boxID))
				var s strings.Builder
				dumpFileListing(&s, b.getFilePath("/"), "/", "", true)
				b.logger.WarnContext(
					ctx,
					"Exit code 127 in box",
					slog.Int("box_id", b.boxID),
					slog.Any("conf", conf),
					slog.Any("command", command),
					slog.String("internal_message", meta.InternalMessage),
					slog.String("fileListing", s.String()), // TODO: Maybe make a []string of file list?
				)
			}
			return meta, nil
		}

		if i > 1 {
			// Only warn if it comes to the second attempt. First error is often enough in prod
			slog.WarnContext(ctx, "Run error in sandbox, retrying. Check grader.log for more details", slog.Int("box_id", b.boxID), slog.Int("attempt", i), slog.Int("max_retries", runErrRetries))
		}
		b.logger.WarnContext(ctx, "Run error in box, retrying", slog.Int("box_id", b.boxID), slog.Int("attempt", i), slog.Int("max_retries", runErrRetries), slog.Any("err", err), slog.Any("metadata", meta))
		time.Sleep(runErrTimeout)
	}

	return meta, nil
}

var keeperOnce sync.Once

// New returns a new box instance from the specified ID
func New(ctx context.Context, id int, memQuota int64, logger *slog.Logger) (eval.Sandbox, error) {
	keeperOnce.Do(func() {
		if err := InitKeeper(ctx); err != nil {
			slog.ErrorContext(ctx, "Could not initialize keeper", slog.Any("err", err))
			os.Exit(1)
		}
	})
	ret, err := exec.Command(isolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--init").CombinedOutput()
	if strings.HasPrefix(string(ret), "Box already exists") {
		slog.InfoContext(ctx, "Box reset", slog.Int("id", id))
		if out, err := exec.Command(isolatePath, "--cg", fmt.Sprintf("--box-id=%d", id), "--cleanup").CombinedOutput(); err != nil {
			slog.WarnContext(ctx, "Could not clean up sandbox", slog.Any("err", err), slog.String("stdout", string(out)))
		}
		return New(ctx, id, memQuota, logger)
	}

	if strings.Contains(string(ret), "incompatible control group mode") { // Created without --cg
		slog.InfoContext(ctx, "Box reset", slog.Int("id", id))
		if out, err := exec.Command(isolatePath, fmt.Sprintf("--box-id=%d", id), "--cleanup").CombinedOutput(); err != nil {
			slog.WarnContext(ctx, "Could not clean up non-cgroup sandbox", slog.Any("err", err), slog.String("stdout", string(out)))
		}
		return New(ctx, id, memQuota, logger)
	}

	if strings.HasPrefix(string(ret), "Must be started as root") {
		if err := os.Chown(isolatePath, 0, 0); err != nil {
			slog.WarnContext(ctx, "Could not chown root the isolate binary", slog.Any("err", err))
			return nil, err
		}
		return New(ctx, id, memQuota, logger)
	}

	if err != nil {
		return nil, err
	}

	return &IsolateBox{path: strings.TrimSpace(string(ret)), boxID: id, memoryQuota: memQuota, logger: logger}, nil
}

func IsolateVersion() string {
	ret, err := exec.Command(isolatePath, "--version").CombinedOutput()
	if err != nil {
		return "precompiled?"
	}
	line, _, _ := bytes.Cut(ret, []byte{'\n'})
	return strings.TrimPrefix(string(line), "The process isolator ")
}

// parseMetaFile parses a specified meta file
func parseMetaFile(ctx context.Context, r io.Reader, out bytes.Buffer) *eval.RunStats {
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
		case "cg-oom-killed":
			file.MemoryLimitExceeded = true
		case "max-rss", "csw-voluntary", "csw-forced", "cg-enabled":
			continue
		default:
			slog.InfoContext(ctx, "Unknown isolate stat", slog.String("key", key), slog.String("value", val))
			continue
		}
	}

	return file
}
