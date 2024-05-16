package scheduler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/KiloProjects/kilonova/eval"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type BoxFunc func(id int, mem int64, logger *slog.Logger) (eval.Sandbox, error)

var _ eval.BoxScheduler = &BoxManager{}

// BoxManager manages a box with eval-based submissions
type BoxManager struct {
	numConcurrent int64
	concSem       *semaphore.Weighted
	memSem        *semaphore.Weighted

	logger *slog.Logger

	availableIDs chan int

	parentMgr *BoxManager

	boxGenerator BoxFunc
}

func (b *BoxManager) SubRunner(ctx context.Context, numConc int64) (eval.BoxScheduler, error) {
	if err := b.concSem.Acquire(ctx, numConc); err != nil {
		return nil, err
	}

	ids := make(chan int, 3*numConc)
	for i := int64(0); i < numConc; i++ {
		ids <- <-b.availableIDs
	}

	return &BoxManager{
		numConcurrent: numConc,
		concSem:       semaphore.NewWeighted(numConc),
		memSem:        b.memSem,

		logger: b.logger,

		availableIDs: ids,

		parentMgr: b,

		boxGenerator: b.boxGenerator,
	}, nil
}

func (b *BoxManager) NumConcurrent() int64 {
	return b.numConcurrent
}

func (b *BoxManager) GetBox(ctx context.Context, memQuota int64) (eval.Sandbox, error) {
	if b.boxGenerator == nil {
		zap.S().Warn("Empty box generator")
		return nil, errors.New("empty box generator")
	}
	if err := b.concSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	if memQuota > 0 {
		if err := b.memSem.Acquire(ctx, memQuota); err != nil {
			return nil, err
		}
	}
	box, err := b.boxGenerator(<-b.availableIDs, memQuota, b.logger)
	if err != nil {
		return nil, err
	}
	// b.logger.Infof("Acquired box %d", box.GetID())
	return box, nil
}

func (b *BoxManager) ReleaseBox(sb eval.Sandbox) {
	q := sb.MemoryQuota()
	if err := sb.Close(); err != nil {
		zap.S().Warnf("Could not release sandbox %d: %v", sb.GetID(), err)
	}
	// b.logger.Infof("Yielded back box %d", sb.GetID())
	b.availableIDs <- sb.GetID()
	b.memSem.Release(q)
	b.concSem.Release(1)
}

// Close waits for all boxes to finish running
func (b *BoxManager) Close(ctx context.Context) error {
	b.concSem.Acquire(ctx, b.numConcurrent)
	if b.parentMgr != nil {
		for len(b.availableIDs) > 0 {
			b.parentMgr.availableIDs <- <-b.availableIDs
		}
		b.parentMgr.concSem.Release(b.numConcurrent)
	}
	close(b.availableIDs)
	return nil
}

// New creates a new box manager
func New(startingNumber int, count int, maxMemory int64, logger *slog.Logger, boxGenerator BoxFunc) (*BoxManager, error) {

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
		availableIDs:  availableIDs,
		numConcurrent: int64(count),

		logger: logger,

		parentMgr: nil,

		boxGenerator: boxGenerator,
	}
	return bm, nil
}

func CheckCanRun(boxFunc BoxFunc) bool {
	box, err := boxFunc(0, 0, slog.Default())
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

func (mgr *BoxManager) RunBox2(ctx context.Context, req *eval.Box2Request, memQuota int64) (*eval.Box2Response, error) {
	goodCmd, err := eval.MakeGoodCommand(req.Command)
	if err != nil {
		slog.Error("Error running MakeGoodCommand", slog.Any("err", err))
		return nil, err
	}

	box, err := mgr.GetBox(ctx, memQuota)
	if err != nil {
		slog.Warn("Could not get box", slog.Any("err", err))
		return nil, err
	}
	defer mgr.ReleaseBox(box)

	for path, val := range req.InputByteFiles {
		if val.Mode == 0 {
			val.Mode = 0666
		}
		if err := box.WriteFile(path, bytes.NewReader(val.Data), val.Mode); err != nil {
			return nil, err
		}
	}

	for path, val := range req.InputBucketFiles {
		if val.Mode == 0 {
			val.Mode = 0666
		}
		if err := eval.CopyInBox(box, val.Bucket, val.Filename, path, val.Mode); err != nil {
			return nil, err
		}
	}

	stats, err := box.RunCommand(ctx, goodCmd, req.RunConfig)
	if err != nil {
		return nil, err
	}

	resp := &eval.Box2Response{
		Stats:       stats,
		ByteFiles:   make(map[string][]byte),
		BucketFiles: make(map[string]*eval.BucketFile),
	}

	var b bytes.Buffer
	for _, path := range req.OutputByteFiles {
		b.Reset()
		if !box.FileExists(path) {
			continue
		}
		if err := box.ReadFile(path, &b); err != nil {
			return nil, err
		}
		resp.ByteFiles[path] = bytes.Clone(b.Bytes())
	}

	for path, file := range req.OutputBucketFiles {
		if file.Mode == 0 {
			file.Mode = 0666
		}

		pr, pw := io.Pipe()
		go func() {
			err := box.ReadFile(path, pw)
			if err != nil {
				slog.Warn("Error reading box file", slog.Any("err", err))
			}
			pw.Close()
		}()

		if err := file.Bucket.WriteFile(file.Filename, pr, file.Mode); err != nil {
			return nil, err
		}
	}

	return resp, nil
}
