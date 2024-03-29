package scheduler

import (
	"context"
	"errors"

	"github.com/KiloProjects/kilonova/eval"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type BoxFunc func(id int, mem int64, logger *zap.SugaredLogger) (eval.Sandbox, error)

var _ eval.BoxScheduler = &BoxManager{}

// BoxManager manages a box with eval-based submissions
type BoxManager struct {
	numConcurrent int64
	concSem       *semaphore.Weighted
	memSem        *semaphore.Weighted

	logger *zap.SugaredLogger

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
func New(startingNumber int, count int, maxMemory int64, logger *zap.SugaredLogger, boxGenerator BoxFunc) (*BoxManager, error) {

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
	box, err := boxFunc(0, 0, zap.S())
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
