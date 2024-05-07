package box

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"sync"

	"github.com/KiloProjects/kilonova/eval"
)

var _ eval.Sandbox = &StupidSandbox{}

// StupidSandbox can be used for testing.
// NOTE: should not be used in a proper environment. It has no proper memory limit
// And time limits are based on manually killing the program
type StupidSandbox struct {
	mu    sync.Mutex
	path  string
	boxID int

	memoryQuota int64

	logger *slog.Logger
}

func (b *StupidSandbox) GetID() int {
	return b.boxID
}

func (b *StupidSandbox) MemoryQuota() int64 {
	return b.memoryQuota
}

func (b *StupidSandbox) RunCommand(ctx context.Context, command []string, conf *eval.RunConfig) (*eval.RunStats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// TODO: Mounts (for chroot)

	// TODO: Running command

	// TODO: Goroutine to wait for time limit and kill program

	// TODO: Read sysprocattr for runstats

	// TODO: Unmounts

	panic("TODO")
}

func (b *StupidSandbox) Close() error {
	return os.RemoveAll(b.path)
}

// getFilePath returns a path to the file location on disk of a box file
func (b *StupidSandbox) getFilePath(boxpath string) string {
	return path.Join(b.path, boxpath)
}

func NewStupid(boxID int, memoryQuota int64, logger *slog.Logger) (eval.Sandbox, error) {
	dirname := path.Join(os.TempDir(), fmt.Sprintf("stupid-box-%d", boxID))
	// Try to clear existing box first, if it exited without cleanup
	if err := os.RemoveAll(dirname); err != nil {
		return nil, err
	}
	// Then create box root and box "home" directory
	if err := os.MkdirAll(path.Join(dirname, "box"), 0766); err != nil {
		return nil, err
	}
	return &StupidSandbox{
		path:        dirname,
		boxID:       boxID,
		memoryQuota: memoryQuota,
		logger:      logger,
	}, errors.New("stupid sandbox is still work in progress")
}

func (b *StupidSandbox) ReadFile(fpath string, w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return readFile(b.getFilePath(fpath), w)
}

func (b *StupidSandbox) WriteFile(fpath string, r io.Reader, mode fs.FileMode) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return writeFile(b.getFilePath(fpath), r, mode)
}

func (b *StupidSandbox) FileExists(fpath string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return checkFile(b.getFilePath(fpath))
}
