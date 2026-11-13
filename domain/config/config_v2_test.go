package config

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
)

// memStore is an in-memory FlagStore standing in for the database layer.
type memStore struct {
	data map[string]json.RawMessage
}

func newMemStore() *memStore { return &memStore{data: make(map[string]json.RawMessage)} }

func (m *memStore) GetFlags(_ context.Context) (map[string]json.RawMessage, error) {
	return m.data, nil
}

func (m *memStore) SetFlag(_ context.Context, key string, value json.RawMessage) error {
	m.data[key] = value
	return nil
}

// Guards the list-typed flags (frontend.banned_hot_problems) through the store
// persistence path.
func TestSliceFlagRoundTrip(t *testing.T) {
	ctx := context.Background()
	f := GenFlag("test.slice", []int{}, "test slice flag")
	store := newMemStore()

	f.Update([]int{1, 2})
	if err := PersistFlag(ctx, store, "test.slice"); err != nil {
		t.Fatal(err)
	}

	f.Update([]int{})
	err := LoadFlagsFromDB(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Value(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("after load: got %v, want [1 2]", got)
	}
}

// An edit persists only the flag that changed, so a concurrent edit to another
// flag cannot be clobbered by a stale whole-file rewrite.
func TestPersistFlagWritesOneRow(t *testing.T) {
	ctx := context.Background()
	GenFlag("test.one", 1, "first")
	GenFlag("test.two", 2, "second")
	store := newMemStore()

	if err := PersistFlag(ctx, store, "test.one"); err != nil {
		t.Fatal(err)
	}
	if len(store.data) != 1 {
		t.Fatalf("wrote %d rows, want 1: %v", len(store.data), store.data)
	}
	if _, ok := store.data["test.one"]; !ok {
		t.Fatalf("wrong row written: %v", store.data)
	}
}

// A corrupt value, an unknown key and the import sentinel must all be survivable:
// startup keeps going and the affected flag keeps its default.
func TestLoadFlagsFromDBTolerance(t *testing.T) {
	ctx := context.Background()
	f := GenFlag("test.tolerant", 7, "tolerant flag")
	store := newMemStore()
	store.data["test.tolerant"] = json.RawMessage(`"not a number"`)
	store.data["test.gone"] = json.RawMessage(`42`)
	store.data[importSentinel] = json.RawMessage(`true`)

	if err := LoadFlagsFromDB(ctx, store); err != nil {
		t.Fatalf("load must not fail on bad data: %v", err)
	}
	if got := f.Value(); got != 7 {
		t.Fatalf("undecodable value should keep the default, got %v", got)
	}
	if _, ok := store.data["test.gone"]; !ok {
		t.Fatal("unknown key must be left in the store for rollback")
	}
}
