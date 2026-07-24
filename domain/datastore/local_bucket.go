package datastore

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova/internal/fsevict"
	"github.com/dustin/go-humanize"
	"github.com/spf13/afero"
)

var (
	_ slog.LogValuer = (*localBucket)(nil)
	_ Bucket         = (*localBucket)(nil)
)

type localBucket struct {
	rootFS afero.Fs
	name   string

	persistent bool
	cache      bool

	maxSize int64         // Maximum size in bytes. Values < 1024 mean system is off
	maxTTL  time.Duration // Maximum duration before emptying

	lastStatsMu sync.RWMutex
	lastStats   *BucketStats
}

// Persistent is a sanity check flag for important buckets such as the tests bucket
// Such that eviction or cleaning is never performed
func (b *localBucket) Persistent() bool {
	return b.persistent
}

// Cache is true only if the bucket should act like a cache. That is, it can be fully purged using the Reset() method
// It's a safeguard against accidentally removing real data
func (b *localBucket) Cache() bool {
	return b.cache
}

type BucketStats struct {
	// Copied from bucket
	Name       string
	Persistent bool
	Cache      bool
	MaxSize    int64         // Maximum size in bytes.
	MaxTTL     time.Duration // Maximum duration before cleaning up object

	CreatedAt time.Time

	// Actual statistics
	NumItems   int
	OnDiskSize int64
}

func (b *localBucket) Statistics(refresh bool) *BucketStats {
	if !refresh && b.lastStats != nil {
		b.lastStatsMu.RLock()
		defer b.lastStatsMu.RUnlock()
		return b.lastStats
	}
	b.lastStatsMu.Lock()
	defer b.lastStatsMu.Unlock()
	b.lastStats = &BucketStats{
		Name: b.name, Cache: b.cache,
		Persistent: b.persistent, MaxSize: b.maxSize, MaxTTL: b.maxTTL,
	}
	entries, err := b.FileList()
	if err != nil {
		slog.WarnContext(context.Background(), "Couldn't get file listing", slog.Any("err", err))
	}
	for _, entry := range entries {
		b.lastStats.NumItems++
		b.lastStats.OnDiskSize += entry.Size()
	}
	b.lastStats.CreatedAt = time.Now()
	return b.lastStats
}

func (b *localBucket) init() error {
	return b.rootFS.MkdirAll(b.name, 0755)
}

func (b *localBucket) stat(name string) (fs.FileInfo, error) {
	stat, err := b.rootFS.Stat(b.filePath(name) + ".zst")
	if err == nil {
		return stat, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	stat, err = b.rootFS.Stat(b.filePath(name) + ".gz")
	if err == nil {
		return stat, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return b.rootFS.Stat(b.filePath(name))
}

func (b *localBucket) Modtime(name string) (time.Time, error) {
	stat, err := b.stat(name)
	if err != nil {
		return time.Now(), err
	}
	return stat.ModTime(), nil
}

func (b *localBucket) Mode(name string) (fs.FileMode, error) {
	stat, err := b.stat(name)
	if err != nil {
		return 0, err
	}
	return stat.Mode(), nil
}

func (b *localBucket) WriteFile(name string, r io.Reader, mode fs.FileMode) error {
	filename := b.filePath(name)
	if err := b.rootFS.RemoveAll(filename + ".zst"); err != nil {
		return err
	}

	f, err := b.rootFS.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func (b *localBucket) Reader(name string) (io.ReadCloser, error) {
	f, err := b.rootFS.Open(b.filePath(name) + ".zst")
	if err == nil {
		return &zstdFileReader{f, newZstdReader(f)}, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	f, err = b.rootFS.Open(b.filePath(name))
	if err == nil {
		return f, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fs.ErrNotExist
	}

	if _, err := b.rootFS.Stat(b.filePath(name) + ".gz"); err == nil {
		return nil, errors.New("can't open file: gzip support removed")
	}
	return nil, err
}

// ReadSeeker tries to open the given file using the normal reader function. If the output implements ReadSeekCloser,
// then it is used directly. Otherwise, we decompress on the fly into a temp file and return that instead (it will be deleted on Close()).
// TODO: Better caching, maybe some kind of sub-bucket concept?
func (b *localBucket) ReadSeeker(name string) (io.ReadSeekCloser, error) {
	rc, err := b.Reader(name)
	if err != nil {
		return nil, err
	}
	if rsc, ok := rc.(io.ReadSeekCloser); ok {
		return rsc, nil
	}
	slog.DebugContext(context.Background(), "ReadSeeker called on compressed file")
	defer rc.Close()
	f, err := os.CreateTemp("", "bucket-temp-*")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, rc); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	return &deletingClosedFile{f}, nil
}

func (b *localBucket) RemoveFile(name string) error {
	if err := b.rootFS.Remove(b.filePath(name) + ".zst"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := b.rootFS.Remove(b.filePath(name) + ".gz"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := b.rootFS.Remove(b.filePath(name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (b *localBucket) FileList() ([]fs.FileInfo, error) {
	entries, err := afero.ReadDir(b.rootFS, b.name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return entries, err
}

func (b *localBucket) Evictable() bool {
	return !b.persistent && (b.maxSize > 1024 || b.maxTTL > time.Second)
}

func (b *localBucket) RunEvictionPolicy(ctx context.Context, logger *slog.Logger) (int, error) {
	if b.persistent {
		return -1, errors.New("bucket is marked as persistent, refusing to run eviction policy")
	}
	b.lastStatsMu.Lock()
	defer b.lastStatsMu.Unlock()

	res, err := fsevict.Sweep(b.rootFS, b.name, b.maxSize, b.maxTTL)
	if err != nil {
		return res.Deleted, err
	}

	if logger != nil {
		logger.InfoContext(ctx, "Before cleanup", slog.Any("bucket", b),
			slog.Int("object_count", res.Deleted+res.Remaining),
			slog.String("bucket_size", humanize.IBytes(uint64(res.DeletedSize+res.RemainingSize))))
	}

	b.lastStats = &BucketStats{
		Name: b.name, Cache: b.cache,
		Persistent: b.persistent, MaxSize: b.maxSize, MaxTTL: b.maxTTL,
		NumItems: res.Remaining, OnDiskSize: res.RemainingSize,
		CreatedAt: time.Now(),
	}

	if logger != nil {
		logger.InfoContext(ctx, "After cleanup", slog.Any("bucket", b),
			slog.Int("object_count", res.Remaining),
			slog.String("bucket_size", humanize.IBytes(uint64(res.RemainingSize))))
	}

	return res.Deleted, nil
}

func (b *localBucket) ResetCache() error {
	if b.persistent {
		return errors.New("bucket is marked as persistent, refusing to delete")
	}
	if !b.cache {
		return errors.New("bucket is not marked as cache, refusing to delete")
	}
	var errs []error
	entries, err := b.FileList()
	if err != nil {
		slog.WarnContext(context.Background(), "Couldn't get file listing", slog.Any("err", err))
	}
	for _, entry := range entries {
		if err := b.RemoveFile(entry.Name()); err != nil {
			errs = append(errs, err)
		}
	}
	// Refresh stats
	b.Statistics(true)
	return errors.Join(errs...)
}

func (b *localBucket) LogValue() slog.Value {
	if b == nil {
		return slog.Value{}
	}
	return slog.StringValue(b.name)
}

func newBucket(rootFS afero.Fs, name string, cache bool, persistent bool, maxSize int64, maxTTL time.Duration) (*localBucket, error) {
	b := &localBucket{
		rootFS:     rootFS,
		name:       name,
		persistent: persistent,
		cache:      cache,
		maxSize:    maxSize,
		maxTTL:     maxTTL,
	}
	return b, b.init()
}

func (b *localBucket) filePath(name string) string {
	return path.Join(b.name, name)
}

type deletingClosedFile struct {
	*os.File
}

func (f *deletingClosedFile) Close() error {
	defer os.Remove(f.Name())
	return f.File.Close()
}
