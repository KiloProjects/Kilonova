package datastore

import (
	"errors"
	"os"

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

	CompressionLevel int
}

var (
	buckets     = make(map[BucketType]*Bucket)
	initialized = false

	// TODO: Do better...
	bucketData = []bucketDef{
		{
			Name:    BucketTypeSubtests,
			IsCache: false,

			CompressionLevel: NoCompression,
		},
		{
			Name:    BucketTypeTests,
			IsCache: false,

			CompressionLevel: DefaultCompression,
		},
		{
			Name:    BucketTypeAttachments,
			IsCache: true,

			CompressionLevel: NoCompression,
		},
		{
			Name:    BucketTypeAvatars,
			IsCache: true,

			CompressionLevel: NoCompression,
		},
		{
			Name:    BucketTypeCheckers,
			IsCache: true,

			CompressionLevel: NoCompression,
		},
		{
			Name:    BucketTypeCompiles,
			IsCache: false, // Well it kind of is but not really since it's cleaned up in the grader

			CompressionLevel: NoCompression,
		},
	}
)

func init() {
	for _, b := range bucketData {
		buckets[b.Name] = nil
	}
}

func InitBuckets(p string) error {
	if initialized {
		return errors.New("buckets already initialized")
	}
	initialized = true
	if err := os.MkdirAll(p, 0777); err != nil {
		return err
	}
	for _, b := range bucketData {
		bucket, err := NewBucket(p, string(b.Name), b.CompressionLevel, b.IsCache)
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
