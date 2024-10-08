package datastore

import (
	"errors"
	"os"
	"time"

	"go.uber.org/zap"
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

func (t BucketType) Valid() bool {
	return t == BucketTypeTests || t == BucketTypeSubtests ||
		t == BucketTypeAttachments || t == BucketTypeAvatars ||
		t == BucketTypeCheckers || t == BucketTypeCompiles
}

type bucketDef struct {
	Name    BucketType
	IsCache bool

	IsPersistent bool
	MaxSize      int64
	MaxTTL       time.Duration

	UseCompression bool
}

var (
	buckets     = make(map[BucketType]*Bucket)
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

func init() {
	for _, b := range bucketData {
		buckets[b.Name] = nil
	}
}

// TODO: Do not do singletons
func InitBuckets(p string) error {
	if initialized {
		return errors.New("buckets already initialized")
	}
	initialized = true
	if err := os.MkdirAll(p, 0777); err != nil {
		return err
	}
	for _, b := range bucketData {
		bucket, err := NewBucket(p, string(b.Name), b.UseCompression, b.IsCache, b.IsPersistent, b.MaxSize, b.MaxTTL)
		if err != nil {
			return err
		}
		buckets[b.Name] = bucket
	}
	return nil
}

func IsBucket(name BucketType) bool {
	_, ok := buckets[name]
	return ok
}

// GetBucket panics if there is no bucket with that name

func GetBucket(name BucketType) *Bucket {
	b, ok := buckets[name]
	if !ok {
		zap.S().Fatalf("No bucket found with name %q", name)
	}
	return b
}

func GetBuckets() []*Bucket {
	ret := make([]*Bucket, 0, len(buckets))
	for _, val := range buckets {
		ret = append(ret, val)
	}
	return ret
}
