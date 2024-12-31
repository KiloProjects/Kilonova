package datastore

import (
	"errors"
	"github.com/KiloProjects/kilonova"
	"os"
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

			IsPersistent:   false,
			UseCompression: false,
		},
		{
			Name:    BucketTypeTests,
			IsCache: false,

			IsPersistent:   true,
			UseCompression: true,
		},
		{
			Name:    BucketTypeAttachments,
			IsCache: true,

			IsPersistent:   false,
			UseCompression: false,
		},
		{
			Name:    BucketTypeAvatars,
			IsCache: true,

			MaxTTL:         31 * 24 * time.Hour, // 31d
			IsPersistent:   false,
			UseCompression: false,
		},
		{
			Name:    BucketTypeCheckers,
			IsCache: true,

			IsPersistent:   false,
			UseCompression: false,
		},
		{
			Name:    BucketTypeCompiles,
			IsCache: false, // Well it kind of is but not really since it's cleaned up in the grader

			IsPersistent:   false,
			UseCompression: false,
		},
	}
)

type bucketDef struct {
	Name    BucketType
	IsCache bool

	IsPersistent bool
	MaxSize      int64
	MaxTTL       time.Duration

	UseCompression bool
}

type Manager struct {
	buckets map[BucketType]*Bucket
}

func (m *Manager) Tests() *Bucket {
	return m.buckets[BucketTypeTests]
}

func (m *Manager) Subtests() *Bucket {
	return m.buckets[BucketTypeSubtests]
}

func (m *Manager) Attachments() *Bucket {
	return m.buckets[BucketTypeAttachments]
}

func (m *Manager) Avatars() *Bucket {
	return m.buckets[BucketTypeAvatars]
}

func (m *Manager) Checkers() *Bucket {
	return m.buckets[BucketTypeCheckers]
}

func (m *Manager) Compilations() *Bucket {
	return m.buckets[BucketTypeCompiles]
}

func (m *Manager) Get(bt BucketType) (*Bucket, error) {
	switch bt {
	case BucketTypeTests, BucketTypeSubtests, BucketTypeAttachments, BucketTypeAvatars, BucketTypeCheckers, BucketTypeCompiles:
		return m.buckets[bt], nil
	default:
		return nil, kilonova.ErrNotFound
	}
}

//func (m *Manager) MustGet(bt BucketType) *Bucket {
//	bucket, err := m.Get(bt)
//	if err != nil {
//		slog.ErrorContext(context.Background(), "No bucket found", slog.Any("type", bt))
//		panic("No bucket found")
//	}
//	return bucket
//}

func (m *Manager) GetAll() (buckets []*Bucket) {
	buckets = make([]*Bucket, 0, len(m.buckets))
	for _, val := range m.buckets {
		buckets = append(buckets, val)
	}
	return
}

func New(rootPath string) (*Manager, error) {
	if initialized {
		return nil, errors.New("buckets already initialized")
	}
	initialized = true
	if err := os.MkdirAll(rootPath, 0777); err != nil {
		return nil, err
	}
	buckets := make(map[BucketType]*Bucket)
	for _, b := range bucketData {
		bucket, err := newBucket(rootPath, string(b.Name), b.UseCompression, b.IsCache, b.IsPersistent, b.MaxSize, b.MaxTTL)
		if err != nil {
			return nil, err
		}
		buckets[b.Name] = bucket
	}
	return &Manager{buckets}, nil
}
