package scheduler

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova/domain/config"
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

// fakeBox stands in for a sandbox: user commands exit at once, the manager
// needs a little longer and reports being interrupted if its context dies first.
type fakeBox struct {
	id  int
	mem int64
}

func (b *fakeBox) ReadFile(string, io.Writer) error                { return nil }
func (b *fakeBox) SaveFile(string, eval.SaveFiler) (string, error) { return "", nil }
func (b *fakeBox) WriteFile(string, io.Reader, fs.FileMode) error  { return nil }
func (b *fakeBox) FileExists(string) bool                          { return false }
func (b *fakeBox) GetID() int                                      { return b.id }
func (b *fakeBox) MemoryQuota() int64                              { return b.mem }
func (b *fakeBox) Close() error                                    { return nil }

func (b *fakeBox) RunCommand(ctx context.Context, cmd []string, _ *eval.RunConfig) (*eval.RunStats, error) {
	if cmd[0] != "/box/manager" {
		return &eval.RunStats{}, nil
	}
	select {
	case <-ctx.Done():
		return &eval.RunStats{Status: "SG", Message: "Interrupted"}, nil
	case <-time.After(50 * time.Millisecond):
		return &eval.RunStats{}, nil
	}
}

// TestMultiboxManagerOutlivesUsers: the manager still has work to do (read the
// last answers, write its verdict) once the user processes are gone, so their
// exit alone must not interrupt it.
func TestMultiboxManagerOutlivesUsers(t *testing.T) {
	config.SetLogToFile(false) // the run audit log would otherwise create a logs dir here
	sched, err := New(0, 2, 1024, slog.Default(), nil, func(_ context.Context, id int, mem int64, _ *slog.Logger) (eval.Sandbox, error) {
		return &fakeBox{id: id, mem: mem}, nil
	})
	require.NoError(t, err)

	resp, _, err := sched.RunMultibox3(t.Context(), &eval.Multibox3Request{
		ManagerSandbox:     &eval.Box3Request{Command: []string{"/box/manager"}, RunConfig: &eval.RunConfig{}},
		UserSandboxConfigs: []*eval.Box3Request{{Command: []string{"/box/user"}, RunConfig: &eval.RunConfig{}}},
		UseStdin:           true,
	}, 1, 1)
	require.NoError(t, err)
	require.Empty(t, resp.Stats.Status)
}
