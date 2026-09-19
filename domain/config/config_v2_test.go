package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
// persistence path and the KN_FLAG_OVERRIDES path.
func TestSliceFlagRoundTrip(t *testing.T) {
	ctx := context.Background()
	f := GenFlag("test.slice", []int{}, "test slice flag")
	store := newMemStore()

	f.Update([]int{1, 2})
	if err := PersistFlag(ctx, store, "test.slice"); err != nil {
		t.Fatal(err)
	}

	f.Update([]int{})
	empty, err := LoadFlagsFromDB(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("store reported empty after a write")
	}
	if got := f.Value(); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("after load: got %v, want [1 2]", got)
	}

	t.Setenv("KN_FLAG_OVERRIDES", "test.slice=[3]")
	ApplyFlagOverrides(ctx)
	if got := f.Value(); !slices.Equal(got, []int{3}) {
		t.Fatalf("after override: got %v, want [3]", got)
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

	if _, err := LoadFlagsFromDB(ctx, store); err != nil {
		t.Fatalf("load must not fail on bad data: %v", err)
	}
	if got := f.Value(); got != 7 {
		t.Fatalf("undecodable value should keep the default, got %v", got)
	}
	if _, ok := store.data["test.gone"]; !ok {
		t.Fatal("unknown key must be left in the store for rollback")
	}
}

// The legacy file is an import source only: it is read, never written or created.
func TestImportFlagsFileLeavesFileAlone(t *testing.T) {
	ctx := context.Background()
	f := GenFlag("test.imported", "default", "imported flag")
	store := newMemStore()

	p := filepath.Join(t.TempDir(), "flags.json")
	contents := []byte(`{"test.imported": "from file"}`)
	if err := os.WriteFile(p, contents, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ImportFlagsFile(ctx, store, p); err != nil {
		t.Fatal(err)
	}
	if got := f.Value(); got != "from file" {
		t.Fatalf("import did not apply the file: got %q", got)
	}
	if _, ok := store.data[importSentinel]; !ok {
		t.Fatal("import sentinel not written")
	}

	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(contents) {
		t.Fatalf("file was rewritten:\n got %s\nwant %s", after, contents)
	}
}

// A missing file must not fail startup, and must not be created.
func TestImportFlagsFileMissing(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	p := filepath.Join(t.TempDir(), "flags.json")

	if err := ImportFlagsFile(ctx, store, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("flags file must not be created, stat err: %v", err)
	}
	if _, ok := store.data[importSentinel]; !ok {
		t.Fatal("sentinel must be written even with no file, so import runs once")
	}
}
