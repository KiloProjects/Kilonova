package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/stretchr/testify/require"
)

// TestGetBoxFailedInitKeepsCapacity: a sandbox that fails to initialise must
// give back its box ID and memory quota, or every failure shrinks the grader.
func TestGetBoxFailedInitKeepsCapacity(t *testing.T) {
	errInit := errors.New("init failed")
	sched, err := New(0, 1, 1024, slog.Default(), nil, func(context.Context, int, int64, *slog.Logger) (eval.Sandbox, error) {
		return nil, errInit
	})
	require.NoError(t, err)
	mgr := sched.(*BoxManager)
	ids := len(mgr.availableIDs)

	// Without the release, the second call blocks on the memory budget until the deadline.
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	for range 2*ids + 1 {
		_, err := mgr.getBox(ctx, 1024)
		require.ErrorIs(t, err, errInit)
	}
	require.Len(t, mgr.availableIDs, ids)
}
