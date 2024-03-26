package datastore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/KiloProjects/kilonova"
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
	stat, err := os.Stat(b.filePath(name) + ".gz")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return os.Stat(b.filePath(name))
		}
		return nil, err
	}
	return stat, nil
}

func (b *Bucket) Writer(name string, mode fs.FileMode) (io.WriteCloser, error) {
	filename := b.filePath(name)
	if b.CompressionLevel != NoCompression {
		filename += ".gz"
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return nil, err
	}
	if b.CompressionLevel != NoCompression {
		return &gzipFileWriter{f, newGzipWriter(f)}, nil
	}
	return f, nil
}

func (b *Bucket) WriteFile(name string, r io.Reader, mode fs.FileMode) error {
	wr, err := b.Writer(name, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, r)
	if err1 := wr.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func (b *Bucket) Reader(name string) (io.ReadCloser, error) {
	f, err := os.Open(b.filePath(name) + ".gz")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			f, err := os.Open(b.filePath(name))
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil, kilonova.ErrNotExist
				}
				return nil, err
			}
			return f, nil
		}
		return nil, err
	}
	gz, err := newGzipReader(f)
	if err != nil {
		return nil, err
	}
	return &gzipFileReader{f, gz}, nil
}

// ReadSeeker opens a new readseeker of the specified file. If uncompressed, it returns the file directly.
// If compressed, then the contents are uncompressed on the fly into a file and that file is then served (it will be deleted on Close()).
// TODO: Better caching, maybe some kind of sub-bucket concept?
func (b *Bucket) ReadSeeker(name string) (io.ReadSeekCloser, error) {
	f, err := os.Open(b.filePath(name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			f, err = os.Open(b.filePath(name) + ".gz")
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil, kilonova.ErrNotExist
				}
				return nil, err
			}
			f2 := &deletingClosedFile{f}
			r, err := newGzipReader(f2)
			if err != nil {
				f2.Close()
				return nil, err
			}
			if _, err := io.Copy(f2, r); err != nil {
				f2.Close()
				return nil, err
			}
			if _, err := f2.Seek(0, io.SeekStart); err != nil {
				f2.Close()
				return nil, err
			}
			return f2, nil
		}
		return nil, err
	}
	return f, nil
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
	err1 := os.Remove(b.filePath(name) + ".gz")
	if err1 != nil && !errors.Is(err1, fs.ErrNotExist) {
		return err1
	}
	err2 := os.Remove(b.filePath(name))
	if err2 != nil && !errors.Is(err2, fs.ErrNotExist) {
		return err2
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
