package datastore

import (
	"cmp"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"slices"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/dustin/go-humanize"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
)

var _ slog.LogValuer = &Bucket{}

type Bucket struct {
	RootPath string
	Name     string

	// Persistent is a sanity check flag for important buckets such as the tests bucket
	// Such that eviction or cleaning is never performed
	Persistent bool

	// Cache is true only if the bucket should act like a cache
	// That is, it can be fully purged using the Reset() method
	// It's a safeguard against accidentally removing real data
	Cache bool

	MaxSize int64         // Maximum size in bytes. Values < 1024 mean system is off
	MaxTTL  time.Duration // Maximum duration before emptying

	// 0 = flate.NoCompression
	// -1 = flate.DefaultCompression
	CompressionLevel int

	lastStatsMu sync.RWMutex
	lastStats   *BucketStats
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

func (b *Bucket) Statistics(refresh bool) *BucketStats {
	if !refresh && b.lastStats != nil {
		b.lastStatsMu.RLock()
		defer b.lastStatsMu.RUnlock()
		return b.lastStats
	}
	b.lastStatsMu.Lock()
	defer b.lastStatsMu.Unlock()
	b.lastStats = &BucketStats{
		Name: b.Name, Cache: b.Cache,
		Persistent: b.Persistent, MaxSize: b.MaxSize, MaxTTL: b.MaxTTL,
	}
	entries, err := b.FileList()
	if err != nil {
		zap.S().Warn(err)
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			zap.S().Warn(err)
			return nil
		}
		b.lastStats.NumItems++
		b.lastStats.OnDiskSize += info.Size()
	}
	b.lastStats.CreatedAt = time.Now()
	return b.lastStats
}

func (b *Bucket) Init() error {
	return os.MkdirAll(path.Join(b.RootPath, b.Name), 0755)
}

func (b *Bucket) Stat(name string) (fs.FileInfo, error) {
	stat, err := os.Stat(b.filePath(name) + ".zst")
	if err == nil {
		return stat, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	stat, err = os.Stat(b.filePath(name) + ".gz")
	if err == nil {
		return stat, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return os.Stat(b.filePath(name))
}

func (b *Bucket) WriteFile(name string, r io.Reader, mode fs.FileMode) error {
	filename := b.filePath(name)
	if b.CompressionLevel != NoCompression {
		filename += ".zst"
	}

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if b.CompressionLevel == NoCompression {
		_, err = io.Copy(f, r)
		if err1 := f.Close(); err1 != nil && err == nil {
			err = err1
		}
		return err
	}

	zw, err := zstd.NewWriter(f, zstd.WithEncoderConcurrency(1))
	if err != nil {
		f.Close()
		return err
	}

	_, err = io.Copy(zw, r)
	if err1 := zw.Close(); err1 != nil && err == nil {
		err = err1
	}
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func (b *Bucket) Reader(name string) (io.ReadCloser, error) {
	f, err := os.Open(b.filePath(name) + ".zst")
	if err == nil {
		return &zstdFileReader{f, newZstdReader(f)}, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	f, err = os.Open(b.filePath(name) + ".gz")
	if err == nil {
		gz, err := newGzipReader(f)
		if err != nil {
			return nil, err
		}
		return &gzipFileReader{f, gz}, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	f, err = os.Open(b.filePath(name))
	if err == nil {
		return f, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, kilonova.ErrNotExist
	}
	return nil, err
}

// ReadSeeker tries to open the given file using the normal reader function. If the output implements ReadSeekCloser,
// then it is used directly. Otherwise, we decompress on the fly into a temp file and return that instead (it will be deleted on Close()).
// TODO: Better caching, maybe some kind of sub-bucket concept?
func (b *Bucket) ReadSeeker(name string) (io.ReadSeekCloser, error) {
	rc, err := b.Reader(name)
	if err != nil {
		return nil, err
	}
	if rsc, ok := rc.(io.ReadSeekCloser); ok {
		return rsc, nil
	}
	zap.S().Debug("ReadSeeker called on compressed file")
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

func (b *Bucket) RemoveFile(name string) error {
	if err := os.Remove(b.filePath(name) + ".zst"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.Remove(b.filePath(name) + ".gz"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.Remove(b.filePath(name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (b *Bucket) FileList() ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(path.Join(b.RootPath, b.Name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return entries, err
}

type evictionEntry struct {
	name    string
	modTime time.Time
	size    int64
}

func (b *Bucket) Evictable() bool {
	return !b.Persistent && (b.MaxSize > 1024 || b.MaxTTL > time.Second)
}

func (b *Bucket) RunEvictionPolicy(logger *slog.Logger) (int, error) {
	if b.Persistent {
		return -1, errors.New("Bucket is marked as persistent, refusing to run eviction policy")
	}
	b.lastStatsMu.Lock()
	defer b.lastStatsMu.Unlock()
	entries, err := os.ReadDir(path.Join(b.RootPath, b.Name))
	if err != nil {
		return -1, err
	}
	var dirSize int64
	// Get directory size and file entries
	evictionEntries := make([]evictionEntry, len(entries))
	for i := range entries {
		info, err := entries[i].Info()
		if err != nil {
			zap.S().Warn(err)
			return -1, nil
		}
		evictionEntries[i].name = info.Name()
		evictionEntries[i].modTime = info.ModTime()
		evictionEntries[i].size = info.Size()
		dirSize += info.Size()
	}
	// Order entries ascending based on file last modified date
	slices.SortFunc(evictionEntries, func(a, b evictionEntry) int {
		return cmp.Compare(a.modTime.UnixMicro(), b.modTime.UnixMicro())
	})

	if logger != nil {
		logger.Info("Before cleanup", slog.Any("bucket", b), slog.Int("object_count", len(evictionEntries)), slog.String("bucket_size", humanize.IBytes(uint64(dirSize))))
	}

	var numDeleted int
	for len(evictionEntries) > 0 {
		var ok bool = true
		// If MaxTTL is big enough and file is earlier than that policy, mark for deletion
		if b.MaxTTL > time.Second && time.Since(evictionEntries[0].modTime) > b.MaxTTL {
			ok = false
		}
		// If directory size is still bigger than maximum
		if b.MaxSize > 1024 && dirSize > b.MaxSize {
			ok = false
		}
		if ok {
			break
		}
		dirSize -= evictionEntries[0].size
		if err := os.Remove(b.filePath(evictionEntries[0].name)); err != nil {
			return numDeleted, err
		}
		numDeleted++
		evictionEntries = evictionEntries[1:]
	}

	b.lastStats = &BucketStats{
		Name: b.Name, Cache: b.Cache,
		Persistent: b.Persistent, MaxSize: b.MaxSize, MaxTTL: b.MaxTTL,
		NumItems: len(evictionEntries), OnDiskSize: dirSize,
		CreatedAt: time.Now(),
	}

	if logger != nil {
		logger.Info("After cleanup", slog.Any("bucket", b), slog.Int("object_count", len(evictionEntries)), slog.String("bucket_size", humanize.IBytes(uint64(dirSize))))
	}

	return numDeleted, nil
}

func (b *Bucket) ResetCache() error {
	if b.Persistent {
		return errors.New("Bucket is marked as persistent, refusing to delete")
	}
	if !b.Cache {
		return errors.New("Bucket is not marked as cache, refusing to delete")
	}
	var errs []error
	entries, err := b.FileList()
	if err != nil {
		zap.S().Warn(err)
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

func (b *Bucket) LogValue() slog.Value {
	return slog.StringValue(b.Name)
}

func NewBucket(path string, name string, compressionLevel int, cache bool, persistent bool, maxSize int64, maxTTL time.Duration) (*Bucket, error) {
	b := &Bucket{
		RootPath:   path,
		Name:       name,
		Persistent: persistent,
		Cache:      cache,
		MaxSize:    maxSize,
		MaxTTL:     maxTTL,

		CompressionLevel: compressionLevel,
	}
	return b, b.Init()
}

func (b *Bucket) filePath(name string) string {
	return path.Join(b.RootPath, b.Name, name)
}

type deletingClosedFile struct {
	*os.File
}

func (f *deletingClosedFile) Close() error {
	defer os.Remove(f.Name())
	return f.File.Close()
}
