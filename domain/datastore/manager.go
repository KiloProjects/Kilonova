package datastore

import (
	"context"
	"errors"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/spf13/afero"

	"io"
	"io/fs"
	"log/slog"
	"time"
)

type BucketType string

const (
	BucketTypeNone        BucketType = ""
	BucketTypeTests       BucketType = "tests"
	BucketTypeSubtests    BucketType = "subtests"
	BucketTypeAttachments BucketType = "attachments"
	BucketTypeAvatars     BucketType = "avatars"
	BucketTypeCheckers    BucketType = "checkers"
	BucketTypeCompiles    BucketType = "compiles"
)

var (
	initialized = false

	// TODO: Do better...
	bucketData = []bucketDef{
		{
			Name:    BucketTypeSubtests,
			IsCache: false,

			MaxSize: 2 * 1024 * 1024 * 1024, // 2GB

			IsPersistent: false,
		},
		{
			Name:    BucketTypeTests,
			IsCache: false,

			IsPersistent: true,
		},
		{
			Name:    BucketTypeAttachments,
			IsCache: true,

			IsPersistent: false,
		},
		{
			Name:    BucketTypeAvatars,
			IsCache: true,

			MaxTTL:       31 * 24 * time.Hour, // 31d
			IsPersistent: false,
		},
		{
			Name:    BucketTypeCheckers,
			IsCache: true,

			IsPersistent: false,
		},
		{
			Name:    BucketTypeCompiles,
			IsCache: false, // Well it kind of is but not really since it's cleaned up in the grader

			IsPersistent: false,
		},
	}
)

type bucketDef struct {
	Name    BucketType
	IsCache bool

	IsPersistent bool
	MaxSize      int64
	MaxTTL       time.Duration
}

type Bucket interface {
	Persistent() bool
	Cache() bool
	Statistics(refresh bool) *BucketStats
	Modtime(name string) (time.Time, error)
	// Deprecated: TODO: Not rely on mode anymore
	Mode(name string) (fs.FileMode, error)
	WriteFile(name string, r io.Reader, mode fs.FileMode) error
	Reader(name string) (io.ReadCloser, error)
	ReadSeeker(name string) (io.ReadSeekCloser, error)
	RemoveFile(name string) error
	FileList() ([]fs.FileInfo, error)
	Evictable() bool
	RunEvictionPolicy(ctx context.Context, logger *slog.Logger) (int, error)
	ResetCache() error
	LogValue() slog.Value
}

type Manager struct {
	buckets map[BucketType]Bucket
}

func (m *Manager) Tests() Bucket {
	return m.buckets[BucketTypeTests]
}

func (m *Manager) Subtests() Bucket {
	return m.buckets[BucketTypeSubtests]
}

func (m *Manager) Attachments() Bucket {
	return m.buckets[BucketTypeAttachments]
}

func (m *Manager) Avatars() Bucket {
	return m.buckets[BucketTypeAvatars]
}

func (m *Manager) Checkers() Bucket {
	return m.buckets[BucketTypeCheckers]
}

func (m *Manager) Compilations() Bucket {
	return m.buckets[BucketTypeCompiles]
}

func (m *Manager) Get(bt BucketType) (Bucket, error) {
	switch bt {
	case BucketTypeTests, BucketTypeSubtests, BucketTypeAttachments, BucketTypeAvatars, BucketTypeCheckers, BucketTypeCompiles:
		return m.buckets[bt], nil
	default:
		return nil, kilonova.ErrNotFound
	}
}

func (m *Manager) GetAll() (buckets []Bucket) {
	buckets = make([]Bucket, 0, len(m.buckets))
	for _, val := range m.buckets {
		buckets = append(buckets, val)
	}
	return
}

func (m *Manager) Reader(bucketType BucketType, name string) (io.ReadCloser, error) {
	bucket, err := m.Get(bucketType)
	if err != nil {
		return nil, err
	}
	return bucket.Reader(name)
}

// Deprecated: No stat should be necessary anymore.
func (m *Manager) Mode(bucketType BucketType, name string) (fs.FileMode, error) {
	bucket, err := m.Get(bucketType)
	if err != nil {
		return 0, fmt.Errorf("error getting bucket: %w", err)
	}
	return bucket.Mode(name)
}

func (m *Manager) WriteFile(bucketType BucketType, name string, r io.Reader, mode fs.FileMode) error {
	bucket, err := m.Get(bucketType)
	if err != nil {
		return err
	}
	return bucket.WriteFile(name, r, mode)
}

func New(rootFS afero.Fs) (*Manager, error) {
	if initialized {
		return nil, errors.New("buckets already initialized")
	}
	initialized = true
	buckets := make(map[BucketType]Bucket)
	for _, b := range bucketData {
		bucket, err := newBucket(rootFS, string(b.Name), b.IsCache, b.IsPersistent, b.MaxSize, b.MaxTTL)
		if err != nil {
			return nil, err
		}
		buckets[b.Name] = bucket
	}
	return &Manager{buckets}, nil
}
