// Package fsevict holds a dependency-light directory eviction sweep shared by
// the datastore buckets and the grader scratch. It imports only afero + stdlib
// so the grader side can use it without dragging in datastore/db.
package fsevict

import (
	"cmp"
	"path"
	"slices"
	"time"

	"github.com/spf13/afero"
)

// Result reports what a Sweep did, enough for callers to log before/after stats.
type Result struct {
	Deleted       int
	Remaining     int
	DeletedSize   int64
	RemainingSize int64
}

type entry struct {
	name    string
	modTime time.Time
	size    int64
}

// Sweep deletes the oldest files in dir (by mtime) until the total size is under
// maxSize and no remaining file is older than maxTTL. A maxSize < 1024 disables
// the size cap; a maxTTL <= time.Second disables the TTL cap (matching the
// datastore bucket semantics it was extracted from).
func Sweep(fsys afero.Fs, dir string, maxSize int64, maxTTL time.Duration) (Result, error) {
	infos, err := afero.ReadDir(fsys, dir)
	if err != nil {
		return Result{}, err
	}
	entries := make([]entry, len(infos))
	var dirSize int64
	for i, info := range infos {
		entries[i] = entry{name: info.Name(), modTime: info.ModTime(), size: info.Size()}
		dirSize += info.Size()
	}
	// Oldest first.
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Compare(a.modTime.UnixMicro(), b.modTime.UnixMicro())
	})

	var res Result
	for len(entries) > 0 {
		ttlExpired := maxTTL > time.Second && time.Since(entries[0].modTime) > maxTTL
		overSize := maxSize > 1024 && dirSize > maxSize
		if !ttlExpired && !overSize {
			break
		}
		if err := fsys.Remove(path.Join(dir, entries[0].name)); err != nil {
			res.Remaining = len(entries)
			res.RemainingSize = dirSize
			return res, err
		}
		dirSize -= entries[0].size
		res.Deleted++
		res.DeletedSize += entries[0].size
		entries = entries[1:]
	}
	res.Remaining = len(entries)
	res.RemainingSize = dirSize
	return res, nil
}
