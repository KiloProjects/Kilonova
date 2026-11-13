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
	store := newMemStore()
	f := GenFlag("test.slice", []int{}, "test slice flag")

	fs := NewFlagset()
	require.NoError(t, fs.InitializeFlags(t.Context(), store))
	require.NoError(t, f.Update(WithFlagset(t.Context(), fs), []int{1, 2}))

	restarted := NewFlagset()
	require.NoError(t, restarted.InitializeFlags(t.Context(), store))
	require.Equal(t, []int{1, 2}, f.Of(restarted))
}

// An edit persists only the flag that changed, so a concurrent edit to another
// flag cannot be clobbered by a stale whole-file rewrite.
func TestUpdateWritesOneRow(t *testing.T) {
	store := newMemStore()
	one := GenFlag("test.one", 1, "first")
	GenFlag("test.two", 2, "second")
	fs := NewFlagset()
	require.NoError(t, fs.InitializeFlags(t.Context(), store))

	require.NoError(t, one.Update(WithFlagset(t.Context(), fs), 5))
	require.Equal(t, map[string]json.RawMessage{"test.one": json.RawMessage(`5`)}, store.data)
}

// The point of flagsets: two instances share definitions but not values or stores.
func TestFlagsetsAreIsolated(t *testing.T) {
	f := GenFlag("test.isolated", "default", "isolated flag")
	storeA, storeB := newMemStore(), newMemStore()
	a, b := NewFlagset(), NewFlagset()
	require.NoError(t, a.InitializeFlags(t.Context(), storeA))
	require.NoError(t, b.InitializeFlags(t.Context(), storeB))

	require.NoError(t, f.Update(WithFlagset(t.Context(), a), "tenant a"))
	require.Equal(t, "tenant a", f.Get(WithFlagset(t.Context(), a)))
	require.Equal(t, "default", f.Of(b))
	require.Equal(t, "default", f.Value(), "global flagset must be untouched")
	require.Empty(t, storeB.data)
}

// A corrupt value and an unknown key must both be survivable:
// startup keeps going and the affected flag keeps its default.
func TestInitializeFlagsTolerance(t *testing.T) {
	store := newMemStore()
	store.data["test.tolerant"] = json.RawMessage(`"not a number"`)
	store.data["test.gone"] = json.RawMessage(`42`)

	f := GenFlag("test.tolerant", 7, "tolerant flag")
	fs := NewFlagset()
	require.NoError(t, fs.InitializeFlags(t.Context(), store), "load must not fail on bad data")
	require.Equal(t, 7, f.Of(fs), "undecodable value should keep the default")
	require.Contains(t, store.data, "test.gone", "unknown key must be left in the store for rollback")
}

// Update, reads and InitializeFlags all touch the flagset; run them together under -race.
func TestConcurrentUpdateAndLoad(t *testing.T) {
	f := GenFlag("test.concurrent", 0, "concurrent flag")
	store := newMemStore()
	store.data["test.concurrent"] = json.RawMessage(`-1`)
	fs := NewFlagset()
	ctx := WithFlagset(t.Context(), fs)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() { require.NoError(t, f.Update(ctx, i)) })
		wg.Go(func() { _ = f.Get(ctx) })
	}
	wg.Go(func() { require.NoError(t, fs.InitializeFlags(ctx, store)) })
	wg.Wait()
}
