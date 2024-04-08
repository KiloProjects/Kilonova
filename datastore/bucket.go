package datastore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
)

type Bucket struct {
	RootPath string
	Name     string

	// Cache is true only if the bucket should act like a cache
	// That is, it can be fully purged using the Reset() method
	// It's a safeguard against accidentally removing real data
	Cache bool

	// 0 = flate.NoCompression
	// -1 = flate.DefaultCompression
	CompressionLevel int

	lastStats    *BucketStats
	lastStatTime time.Time
}

type BucketStats struct {
	Name  string
	Cache bool

	NumItems   int
	OnDiskSize int64
}

func (b *Bucket) Statistics() *BucketStats {
	if time.Since(b.lastStatTime) > 1*time.Minute {
		b.lastStats = &BucketStats{Name: b.Name, Cache: b.Cache}
		b.IterFiles(func(entry fs.DirEntry) error {
			info, err := entry.Info()
			if err != nil {
				zap.S().Warn(err)
				return nil
			}
			b.lastStats.NumItems++
			b.lastStats.OnDiskSize += info.Size()
			return nil
		})
	}
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
		err1 = err
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

func (b *Bucket) IterFiles(f func(entry fs.DirEntry) error) error {
	entries, err := os.ReadDir(path.Join(b.RootPath, b.Name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if err := f(entry); err != nil {
			return err
		}
	}

	return nil
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

func (b *Bucket) ResetCache() error {
	if !b.Cache {
		return errors.New("Bucket is not marked as cache, refusing to delete")
	}
	var errs []error
	b.IterFiles(func(entry fs.DirEntry) error {
		if err := b.RemoveFile(entry.Name()); err != nil {
			errs = append(errs, err)
		}
		return nil
	})
	b.lastStatTime = time.Time{}
	return errors.Join(errs...)
}

func NewBucket(path string, name string, compressionLevel int, cache bool) (*Bucket, error) {
	b := &Bucket{path, name, cache, compressionLevel, nil, time.Time{}}
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
