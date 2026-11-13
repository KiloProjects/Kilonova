package config

import (
	"context"
	"encoding/json"
	"maps"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// memStore is an in-memory FlagStore standing in for the database layer.
type memStore struct {
	mu   sync.Mutex
	data map[string]json.RawMessage
}

func newMemStore() *memStore { return &memStore{data: make(map[string]json.RawMessage)} }

func (m *memStore) GetFlags(_ context.Context) (map[string]json.RawMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.data), nil
}

func (m *memStore) SetFlag(_ context.Context, key string, value json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

// Guards the list-typed flags (frontend.banned_hot_problems) through the store
// persistence path: an Update must survive a "restart" into a fresh flagset.
func TestSliceFlagRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := newMemStore()

	fs := newFlagset()
	f := fs.GenFlag("test.slice", []int{}, "test slice flag")
	require.NoError(t, fs.InitializeFlags(ctx, store))
	f.Update(ctx, []int{1, 2})

	restarted := newFlagset()
	f2 := restarted.GenFlag("test.slice", []int{}, "test slice flag")
	require.NoError(t, restarted.InitializeFlags(ctx, store))
	require.Equal(t, []int{1, 2}, f2.Value())
}

// An edit persists only the flag that changed, so a concurrent edit to another
// flag cannot be clobbered by a stale whole-file rewrite.
func TestUpdateWritesOneRow(t *testing.T) {
	ctx := t.Context()
	store := newMemStore()
	fs := newFlagset()
	one := fs.GenFlag("test.one", 1, "first")
	fs.GenFlag("test.two", 2, "second")
	require.NoError(t, fs.InitializeFlags(ctx, store))

	one.Update(ctx, 5)
	require.Equal(t, map[string]json.RawMessage{"test.one": json.RawMessage(`5`)}, store.data)
}

// A corrupt value and an unknown key must both be survivable:
// startup keeps going and the affected flag keeps its default.
func TestInitializeFlagsTolerance(t *testing.T) {
	ctx := t.Context()
	store := newMemStore()
	store.data["test.tolerant"] = json.RawMessage(`"not a number"`)
	store.data["test.gone"] = json.RawMessage(`42`)

	fs := newFlagset()
	f := fs.GenFlag("test.tolerant", 7, "tolerant flag")
	require.NoError(t, fs.InitializeFlags(ctx, store), "load must not fail on bad data")
	require.Equal(t, 7, f.Value(), "undecodable value should keep the default")
	require.Contains(t, store.data, "test.gone", "unknown key must be left in the store for rollback")
}

// Update and InitializeFlags take the flagset and flag locks; run them together
// under -race to catch a lock-order inversion or an unguarded store read.
func TestConcurrentUpdateAndLoad(t *testing.T) {
	ctx := t.Context()
	fs := newFlagset()
	f := fs.GenFlag("test.concurrent", 0, "concurrent flag")

	store := newMemStore()
	store.data["test.concurrent"] = json.RawMessage(`-1`) // so InitializeFlags takes the flag lock too

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() { f.Update(ctx, i) })
		wg.Go(func() { _ = f.Value() })
	}
	wg.Go(func() { require.NoError(t, fs.InitializeFlags(ctx, store)) })
	wg.Wait()
}
