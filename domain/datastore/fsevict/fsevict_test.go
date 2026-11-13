package fsevict

import (
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, fsys afero.Fs, name string, size int, age time.Duration) {
	t.Helper()
	require.NoError(t, afero.WriteFile(fsys, name, make([]byte, size), 0o644))
	mod := time.Now().Add(-age)
	require.NoError(t, fsys.Chtimes(name, mod, mod))
}

func exists(t *testing.T, fsys afero.Fs, name string) bool {
	t.Helper()
	ok, err := afero.Exists(fsys, name)
	require.NoError(t, err)
	return ok
}

func TestSweepTTL(t *testing.T) {
	fsys := afero.NewMemMapFs()
	write(t, fsys, "old", 10, 2*time.Hour)
	write(t, fsys, "young", 10, time.Minute)

	res, err := Sweep(fsys, ".", 0, time.Hour)
	require.NoError(t, err)
	require.Equal(t, 1, res.Deleted)
	require.Equal(t, 1, res.Remaining)
	assert.False(t, exists(t, fsys, "old"), "expired file was not deleted")
	assert.True(t, exists(t, fsys, "young"), "young file was deleted by TTL sweep")
}

func TestSweepSize(t *testing.T) {
	fsys := afero.NewMemMapFs()
	// Three 2000-byte files, cap 2500: oldest two must go, newest (2000) stays under.
	write(t, fsys, "a", 2000, 3*time.Minute)
	write(t, fsys, "b", 2000, 2*time.Minute)
	write(t, fsys, "c", 2000, time.Minute)

	res, err := Sweep(fsys, ".", 2500, 0)
	require.NoError(t, err)
	require.Equal(t, 2, res.Deleted)
	require.Equal(t, 1, res.Remaining)
	assert.True(t, exists(t, fsys, "c"), "newest file evicted while over size — LRU order wrong")
	assert.False(t, exists(t, fsys, "a"), "oldest files not evicted to meet size cap")
	assert.False(t, exists(t, fsys, "b"), "oldest files not evicted to meet size cap")
}

func TestSweepDisabledCapsKeepEverything(t *testing.T) {
	fsys := afero.NewMemMapFs()
	write(t, fsys, "old", 10, 100*time.Hour)
	// maxSize < 1024 and maxTTL <= 1s both disable the caps.
	res, err := Sweep(fsys, ".", 0, 0)
	require.NoError(t, err)
	require.Zero(t, res.Deleted, "disabled caps must not delete anything")
	require.True(t, exists(t, fsys, "old"), "disabled caps must not delete anything")
}
