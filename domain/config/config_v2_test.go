package config

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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
	require.Equal(t, "default", f.Get(t.Context()), "a ctx without a flagset reads the default")
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
		wg.Go(func() { assert.NoError(t, f.Update(ctx, i)) })
		wg.Go(func() { _ = f.Get(ctx) })
	}
	wg.Go(func() { assert.NoError(t, fs.InitializeFlags(ctx, store)) })
	wg.Wait()
}

func TestSecretMarking(t *testing.T) {
	require.True(t, GenSecretFlag("test.secret.marked", "", "secret").Secret())
	require.False(t, GenFlag("test.secret.plain", "", "plain").Secret())
}

func TestFlagNamesSorted(t *testing.T) {
	GenFlag("test.names.b", 0, "b")
	GenFlag("test.names.a", 0, "a")
	names := FlagNames()
	require.True(t, slices.IsSorted(names))
	require.Less(t, slices.Index(names, "test.names.a"), slices.Index(names, "test.names.b"))
	require.NotEqual(t, -1, slices.Index(names, "test.names.a"))
}

func TestExport(t *testing.T) {
	set := GenFlag("test.export.set", 1, "set")
	GenFlag("test.export.default", 2, "default")
	secret := GenSecretFlag("test.export.secret", "", "secret")
	fs := NewFlagset()
	ctx := WithFlagset(t.Context(), fs)
	require.NoError(t, set.Update(ctx, 5))
	require.NoError(t, secret.Update(ctx, "hunter2"))

	out, err := fs.Export(false)
	require.NoError(t, err)
	require.Equal(t, map[string]json.RawMessage{"test.export.set": json.RawMessage(`5`)}, out,
		"only explicitly set, non-secret flags without opt-in")

	out, err = fs.Export(true)
	require.NoError(t, err)
	require.Equal(t, json.RawMessage(`"hunter2"`), out["test.export.secret"], "secret included with opt-in")
	require.NotContains(t, out, "test.export.default", "defaults are never pinned")
}

func TestImport(t *testing.T) {
	num := GenFlag("test.import.num", 0, "num")
	list := GenFlag("test.import.list", []string{}, "list")
	untouched := GenFlag("test.import.untouched", "keep", "untouched")
	secret := GenSecretFlag("test.import.secret", "", "secret")
	store := newMemStore()
	fs := NewFlagset()
	require.NoError(t, fs.InitializeFlags(t.Context(), store))
	require.NoError(t, untouched.Update(WithFlagset(t.Context(), fs), "before"))

	res := fs.Import(t.Context(), map[string]json.RawMessage{
		"test.import.num":    json.RawMessage(`"abc"`),
		"test.import.list":   json.RawMessage(`["a","b"]`),
		"test.import.secret": json.RawMessage(`"s3cret"`),
		"test.import.gone":   json.RawMessage(`1`),
	})

	require.Equal(t, []string{"test.import.list", "test.import.secret"}, res.Applied)
	require.Equal(t, []string{"test.import.secret"}, res.AppliedSecrets)
	require.Contains(t, res.Skipped, "test.import.gone")
	require.Contains(t, res.Skipped, "test.import.num", "wrong type is skipped, not fatal")
	require.Empty(t, res.Errors)

	require.Equal(t, 0, num.Of(fs))
	require.Equal(t, []string{"a", "b"}, list.Of(fs))
	require.Equal(t, "s3cret", secret.Of(fs))
	require.Equal(t, "before", untouched.Of(fs), "flags absent from the file are untouched")
	require.Equal(t, json.RawMessage(`["a","b"]`), store.data["test.import.list"], "imported values are persisted")
}

func TestGetFlagWrongType(t *testing.T) {
	GenFlag("test.wrongtype", 1, "int flag")
	f, ok := GetFlag[string]("test.wrongtype")
	require.False(t, ok)
	require.Nil(t, f, "a miss must be a nil interface, not a typed nil")
}
