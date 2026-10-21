package fsevict

import (
	"testing"
	"time"

	"github.com/spf13/afero"
)

func write(t *testing.T, fsys afero.Fs, name string, size int, age time.Duration) {
	t.Helper()
	if err := afero.WriteFile(fsys, name, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	mod := time.Now().Add(-age)
	if err := fsys.Chtimes(name, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func exists(t *testing.T, fsys afero.Fs, name string) bool {
	t.Helper()
	ok, err := afero.Exists(fsys, name)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestSweepTTL(t *testing.T) {
	fsys := afero.NewMemMapFs()
	write(t, fsys, "old", 10, 2*time.Hour)
	write(t, fsys, "young", 10, time.Minute)

	res, err := Sweep(fsys, ".", 0, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 1 || res.Remaining != 1 {
		t.Fatalf("want 1 deleted / 1 remaining, got %+v", res)
	}
	if exists(t, fsys, "old") {
		t.Error("expired file was not deleted")
	}
	if !exists(t, fsys, "young") {
		t.Error("young file was deleted by TTL sweep")
	}
}

func TestSweepSize(t *testing.T) {
	fsys := afero.NewMemMapFs()
	// Three 2000-byte files, cap 2500: oldest two must go, newest (2000) stays under.
	write(t, fsys, "a", 2000, 3*time.Minute)
	write(t, fsys, "b", 2000, 2*time.Minute)
	write(t, fsys, "c", 2000, time.Minute)

	res, err := Sweep(fsys, ".", 2500, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 2 || res.Remaining != 1 {
		t.Fatalf("want 2 deleted / 1 remaining, got %+v", res)
	}
	if !exists(t, fsys, "c") {
		t.Error("newest file evicted while over size — LRU order wrong")
	}
	if exists(t, fsys, "a") || exists(t, fsys, "b") {
		t.Error("oldest files not evicted to meet size cap")
	}
}

func TestSweepDisabledCapsKeepEverything(t *testing.T) {
	fsys := afero.NewMemMapFs()
	write(t, fsys, "old", 10, 100*time.Hour)
	// maxSize < 1024 and maxTTL <= 1s both disable the caps.
	res, err := Sweep(fsys, ".", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 0 || !exists(t, fsys, "old") {
		t.Fatalf("disabled caps must not delete anything, got %+v", res)
	}
}
