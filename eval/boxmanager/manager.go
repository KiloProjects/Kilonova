package boxmanager

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

var _ eval.Runner = &BoxManager{}

// BoxManager manages a box with eval-based submissions
type BoxManager struct {
	dm kilonova.GraderStore

	numConcurrent int
	sem           *semaphore.Weighted

	availableIDs chan int

	// If debug mode is enabled, the manager should print more stuff to the command line
	debug bool
}

// ToggleDebug is a convenience function to setting up debug mode in the box manager and all future boxes
// It should print additional output
func (b *BoxManager) ToggleDebug() {
	b.debug = !b.debug
}

func (b *BoxManager) RunTask(ctx context.Context, task eval.Task) error {
	box, err := b.getSandbox(ctx)
	if err != nil {
		zap.S().Info(err)
		return err
	}
	defer b.releaseSandbox(box)
	return task.Execute(ctx, box)
}

func (b *BoxManager) newSandbox() (*Box, error) {
	box, err := newBox(<-b.availableIDs)
	if err != nil {
		return nil, err
	}
	box.Debug = b.debug
	return box, nil
}

func (b *BoxManager) getSandbox(ctx context.Context) (eval.Sandbox, error) {
	if err := b.sem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	return b.newSandbox()
}

func (b *BoxManager) releaseSandbox(sb eval.Sandbox) {
	if err := sb.Close(); err != nil {
		zap.S().Warnf("Could not release sandbox %d: %v", sb.GetID(), err)
	}
	b.availableIDs <- sb.GetID()
	b.sem.Release(1)
}

// Close waits for all boxes to finish running
func (b *BoxManager) Close(ctx context.Context) error {
	b.sem.Acquire(ctx, int64(b.numConcurrent))
	close(b.availableIDs)
	return nil
}

// New creates a new box manager
func New(count int, dm kilonova.GraderStore) (*BoxManager, error) {

	sem := semaphore.NewWeighted(int64(count))

	availableIDs := make(chan int, 3*count)
	for i := 1; i <= 2*count; i++ {
		availableIDs <- i
	}

	bm := &BoxManager{
		dm:            dm,
		sem:           sem,
		availableIDs:  availableIDs,
		numConcurrent: count,
	}
	return bm, nil
}
