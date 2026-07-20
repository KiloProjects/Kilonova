package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sys/unix"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	cmdAuditLogger = sync.OnceValue(func() *slog.Logger {
		return slog.New(slog.NewJSONHandler(&lumberjack.Logger{
			Filename: path.Join(config.Common.LogDir, "sandbox_runs.log"),
			MaxSize:  200, // MB
			Compress: true,
		}, &slog.HandlerOptions{
			AddSource: false,
		}))
	})
)

type BoxFunc func(ctx context.Context, id int, mem int64, logger *slog.Logger) (eval.Sandbox, error)

var _ eval.Box3Scheduler = &BoxManager{}

// BoxManager manages a box with eval-based submissions
type BoxManager struct {
	numConcurrent int64
	// concSem measures the number of running Box3 requests.
	// Since a request will be able to have multiple boxes (communication type submissions), it does not reflect the number of concurrent boxes running.
	concSem   *semaphore.Weighted
	memSem    *semaphore.Weighted
	maxMemory int64

	logger *slog.Logger

	availableIDs chan int

	boxGenerator BoxFunc

	scratch eval.Scratch
}

func (mgr *BoxManager) getBox(ctx context.Context, memQuota int64) (eval.Sandbox, error) {
	if mgr.boxGenerator == nil {
		slog.WarnContext(ctx, "Empty box generator")
		return nil, errors.New("empty box generator")
	}
	if memQuota > 0 {
		if err := mgr.memSem.Acquire(ctx, memQuota); err != nil {
			return nil, err
		}
	}
	box, err := mgr.boxGenerator(ctx, <-mgr.availableIDs, memQuota, mgr.logger)
	if err != nil {
		return nil, err
	}
	// b.logger.Infof("Acquired box %d", box.GetID())
	return box, nil
}

func (mgr *BoxManager) releaseBox(ctx context.Context, sb eval.Sandbox) {
	q := sb.MemoryQuota()
	if err := sb.Close(); err != nil {
		slog.WarnContext(ctx, "Could not release sandbox", slog.Any("box_id", sb.GetID()), slog.Any("err", err))
	}
	// b.logger.Infof("Yielded back box %d", sb.GetID())
	mgr.availableIDs <- sb.GetID()
	mgr.memSem.Release(q)
}

// Close waits for all boxes to finish running
func (mgr *BoxManager) Close(ctx context.Context) error {
	mgr.concSem.Acquire(ctx, mgr.numConcurrent)
	close(mgr.availableIDs)
	return nil
}

// New creates a new box manager
func New(startingNumber int, count int, maxMemory int64, logger *slog.Logger, scratch eval.Scratch, boxGenerator BoxFunc) (eval.Box3Scheduler, error) {

	if startingNumber < 0 {
		startingNumber = 0
	}

	availableIDs := make(chan int, 3*count)
	for i := 1; i <= 2*count; i++ {
		availableIDs <- i + startingNumber
	}

	bm := &BoxManager{
		concSem:       semaphore.NewWeighted(int64(count)),
		memSem:        semaphore.NewWeighted(maxMemory),
		maxMemory:     maxMemory,
		availableIDs:  availableIDs,
		numConcurrent: int64(count),

		logger: logger,

		boxGenerator: boxGenerator,

		scratch: scratch,
	}
	return bm, nil
}

func CheckCanRun(ctx context.Context, boxFunc BoxFunc) bool {
	box, err := boxFunc(ctx, 0, 0, slog.Default())
	if err != nil {
		slog.WarnContext(ctx, "Error creating sandbox", slog.Any("err", err))
		return false
	}
	if err := box.Close(); err != nil {
		slog.WarnContext(ctx, "Error closing sandbox", slog.Any("err", err))
		return false
	}
	return true
}

func (mgr *BoxManager) RunBox3(ctx context.Context, req *eval.Box3Request, memQuota int64) (*eval.Box3Response, error) {
	goodCmd, err := makeGoodCommand(req)
	if err != nil {
		slog.ErrorContext(ctx, "Error running MakeGoodCommand", slog.Any("err", err))
		return nil, err
	}

	if err := mgr.concSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer mgr.concSem.Release(1)
	box, err := mgr.getBox(ctx, memQuota)
	if err != nil {
		slog.WarnContext(ctx, "Could not get box", slog.Any("err", err))
		return nil, err
	}
	defer mgr.releaseBox(ctx, box)

	if err := mgr.setupSandbox(box, req); err != nil {
		return nil, err
	}

	stats, err := box.RunCommand(ctx, goodCmd, req.RunConfig)
	if err != nil {
		return nil, err
	}
	cmdAuditLogger().InfoContext(ctx, "Ran command",
		slog.Any("command", goodCmd),
		slog.Any("stats", stats),
		slog.Any("output_files", req.OutputFilePaths),
		slog.Int64("mem_quota", memQuota),
	)

	return mgr.collectResponse(box, req, stats)
}

func (mgr *BoxManager) RunMultibox3(ctx context.Context, req *eval.Multibox3Request, managerMemQuota int64, individualMemQuota int64) (*eval.Box3Response, []*eval.RunStats, error) {
	if managerMemQuota+int64(len(req.UserSandboxConfigs))*individualMemQuota > mgr.maxMemory {
		return nil, nil, errors.New("total memory quota exceeds max memory")
	}
	if int64(len(req.UserSandboxConfigs)+1) > mgr.numConcurrent {
		return nil, nil, errors.New("number of sandboxes exceeds max concurrent")
	}

	// Format commands for the sandboxes
	var err error
	req.ManagerSandbox.Command, err = makeGoodCommand(req.ManagerSandbox)
	if err != nil {
		slog.ErrorContext(ctx, "Error running MakeGoodCommand", slog.Any("err", err))
		return nil, nil, err
	}
	for i := range req.UserSandboxConfigs {
		req.UserSandboxConfigs[i].Command, err = makeGoodCommand(req.UserSandboxConfigs[i])
		if err != nil {
			slog.ErrorContext(ctx, "Error running MakeGoodCommand", slog.Any("err", err))
			return nil, nil, err
		}
	}

	// Acquire the semaphores for the manager and the user sandboxes
	if err := mgr.concSem.Acquire(ctx, int64(len(req.UserSandboxConfigs)+1)); err != nil {
		return nil, nil, err
	}
	defer mgr.concSem.Release(int64(len(req.UserSandboxConfigs) + 1))

	// Initialize the communication FIFOs
	fifoDirs := make([]string, len(req.UserSandboxConfigs))
	fifoUserToManager := make([]string, len(req.UserSandboxConfigs))
	fifoManagerToUser := make([]string, len(req.UserSandboxConfigs))
	for i := range fifoDirs {
		dir, err := os.MkdirTemp("", "comm-fifo-*")
		if err != nil {
			return nil, nil, err
		}
		defer os.RemoveAll(dir)

		if err := os.Chmod(dir, 0755); err != nil {
			return nil, nil, err
		}
		fifoDirs[i] = dir

		fifoUserToManager[i] = path.Join(dir, fmt.Sprintf("u%d_to_m", i))
		if err := unix.Mkfifo(fifoUserToManager[i], 0666); err != nil {
			return nil, nil, err
		}
		if err := os.Chmod(fifoUserToManager[i], 0666); err != nil {
			return nil, nil, err
		}
		fifoManagerToUser[i] = path.Join(dir, fmt.Sprintf("m_to_u%d", i))
		if err := unix.Mkfifo(fifoManagerToUser[i], 0666); err != nil {
			return nil, nil, err
		}
		if err := os.Chmod(fifoManagerToUser[i], 0666); err != nil {
			return nil, nil, err
		}
	}
	sandboxFifoDirs := make([]string, len(req.UserSandboxConfigs))
	sandboxFifoUserToManager := make([]string, len(req.UserSandboxConfigs))
	sandboxFifoManagerToUser := make([]string, len(req.UserSandboxConfigs))
	for i := range sandboxFifoDirs {
		sandboxFifoDirs[i] = fmt.Sprintf("/fifo%d", i)
		sandboxFifoUserToManager[i] = path.Join(sandboxFifoDirs[i], fmt.Sprintf("u%d_to_m", i))
		sandboxFifoManagerToUser[i] = path.Join(sandboxFifoDirs[i], fmt.Sprintf("m_to_u%d", i))

		req.ManagerSandbox.RunConfig.Directories = append(req.ManagerSandbox.RunConfig.Directories, language.Directory{
			In:   sandboxFifoDirs[i],
			Out:  fifoDirs[i],
			Opts: "rw",
		})
		req.ManagerSandbox.Command = append(req.ManagerSandbox.Command, sandboxFifoUserToManager[i], sandboxFifoManagerToUser[i])

		req.UserSandboxConfigs[i].RunConfig.Directories = append(req.UserSandboxConfigs[i].RunConfig.Directories, language.Directory{
			In:   sandboxFifoDirs[i],
			Out:  fifoDirs[i],
			Opts: "rw",
		})

		if req.UseStdin {
			req.UserSandboxConfigs[i].RunConfig.InputPath = sandboxFifoManagerToUser[i]
			req.UserSandboxConfigs[i].RunConfig.OutputPath = sandboxFifoUserToManager[i]
		} else {
			req.UserSandboxConfigs[i].Command = append(req.UserSandboxConfigs[i].Command, sandboxFifoManagerToUser[i], sandboxFifoUserToManager[i])
		}

		if len(req.UserSandboxConfigs) > 1 {
			req.UserSandboxConfigs[i].Command = append(req.UserSandboxConfigs[i].Command, strconv.Itoa(i))
		}
	}

	// Initialize the sandboxes
	managerBox, err := mgr.getBox(ctx, managerMemQuota)
	if err != nil {
		slog.WarnContext(ctx, "Could not get box", slog.Any("err", err))
		return nil, nil, err
	}
	defer mgr.releaseBox(ctx, managerBox)

	if err := mgr.setupSandbox(managerBox, req.ManagerSandbox); err != nil {
		return nil, nil, err
	}

	userBoxes := make([]eval.Sandbox, len(req.UserSandboxConfigs))
	for i := range req.UserSandboxConfigs {
		userBoxes[i], err = mgr.getBox(ctx, individualMemQuota)
		if err != nil {
			slog.WarnContext(ctx, "Could not get box", slog.Any("err", err))
			return nil, nil, err
		}
		defer mgr.releaseBox(ctx, userBoxes[i])

		if err := mgr.setupSandbox(userBoxes[i], req.UserSandboxConfigs[i]); err != nil {
			return nil, nil, err
		}
	}

	var wg, userWg sync.WaitGroup
	userStats := make([]*eval.RunStats, len(req.UserSandboxConfigs))
	wg.Add(len(req.UserSandboxConfigs) + 1)
	userWg.Add(len(req.UserSandboxConfigs))

	errChan := make(chan error, len(req.UserSandboxConfigs)+1)
	respChan := make(chan *eval.Box3Response, 1)

	managerCtx, cancelManager := context.WithCancel(ctx)
	userCtx, cancelUser := context.WithCancel(ctx)

	go func() {
		defer wg.Done()
		stats, err := managerBox.RunCommand(managerCtx, req.ManagerSandbox.Command, req.ManagerSandbox.RunConfig)
		if err != nil {
			errChan <- err
			return
		}
		cmdAuditLogger().InfoContext(ctx, "Ran manager command",
			slog.Any("command", req.ManagerSandbox.Command),
			slog.Any("stats", stats),
			slog.Any("output_files", req.ManagerSandbox.OutputFilePaths),
			slog.Int64("mem_quota", managerMemQuota),
		)

		resp, err := mgr.collectResponse(managerBox, req.ManagerSandbox, stats)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- resp
		cancelUser()
	}()

	for i := range req.UserSandboxConfigs {
		go func(i int) {
			defer wg.Done()
			stats, err := userBoxes[i].RunCommand(userCtx, req.UserSandboxConfigs[i].Command, req.UserSandboxConfigs[i].RunConfig)
			if err != nil {
				errChan <- err
				return
			}
			cmdAuditLogger().InfoContext(ctx, "Ran communication user command",
				slog.Any("command", req.UserSandboxConfigs[i].Command),
				slog.Any("stats", stats),
				slog.Int64("mem_quota", individualMemQuota),
			)
			userStats[i] = stats
		}(i)
	}

	go func() {
		userWg.Wait()
		cancelManager()
	}()

	wg.Wait()
	close(errChan)
	close(respChan)

	var errs []error
	for err := range errChan {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, userStats, errors.Join(errs...)
	}

	resp, ok := <-respChan
	if !ok {
		return nil, userStats, errors.New("no response from manager")
	}

	return resp, userStats, nil
}

func (mgr *BoxManager) copyScratchFile(box eval.Sandbox, sf eval.ScratchFile) error {
	rc, err := mgr.scratch.ReadFile(sf.Identifier)
	if err != nil {
		return err
	}
	defer rc.Close()

	return box.WriteFile(sf.BoxPath, rc, sf.Mode)
}

// setupSandbox copies the request files into the sandbox.
func (mgr *BoxManager) setupSandbox(box eval.Sandbox, req *eval.Box3Request) error {
	for _, file := range req.InputFiles {
		if file.Mode == 0 {
			file.Mode = 0666
		}
		if err := mgr.copyScratchFile(box, file); err != nil {
			return err
		}
	}

	return nil
}

func (mgr *BoxManager) collectResponse(box eval.Sandbox, req *eval.Box3Request, stats *eval.RunStats) (*eval.Box3Response, error) {
	resp := &eval.Box3Response{
		Stats: stats,
		Files: make(map[string]string),
	}

	for _, filePath := range req.OutputFilePaths {
		if !box.FileExists(filePath) {
			continue
		}
		identifier, err := box.SaveFile(filePath, mgr.scratch.SaveFile)
		if err != nil {
			return nil, err
		}
		resp.Files[filePath] = identifier
	}

	return resp, nil
}

// makeGoodCommand makes sure it's a full path (with no symlinks) for the command.
// Some languages (like java) are hidden pretty deep in symlinks, and we don't want a hardcoded path that could be different on other platforms.
func makeGoodCommand(req *eval.Box3Request) ([]string, error) {
	tmp := slices.Clone(req.Command)
	if len(tmp) == 0 {
		return nil, errors.New("no command given")
	}

	if strings.HasPrefix(tmp[0], "/box") {
		return tmp, nil
	}

	if strings.HasSuffix(tmp[0], "uv") {
		tmp[0] = "/mnt/uv/bin/uv"
		return tmp, nil
	}

	cmd, err := exec.LookPath(tmp[0])
	if err != nil {
		return nil, err
	}

	cmd2, err := filepath.EvalSymlinks(cmd)
	if err != nil {
		return nil, err
	}
	// Latest fedora fix
	if !strings.Contains(cmd2, "ccache") {
		cmd = cmd2
	}

	tmp[0] = cmd
	return tmp, nil
}
