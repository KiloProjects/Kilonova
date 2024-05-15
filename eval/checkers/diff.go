package checkers

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/KiloProjects/kilonova/datastore"
	"github.com/shopspring/decimal"
)

const (
	ErrOut     = "translate:internal_error"
	CorrectOut = "translate:success"
	WrongOut   = "translate:wrong"
)

var _ Checker = &DiffChecker{}

type DiffChecker struct{}

func (d *DiffChecker) Prepare(_ context.Context) (string, error) { return "", nil }

func (d *DiffChecker) Cleanup(_ context.Context) error { return nil }

func (d *DiffChecker) RunChecker(ctx context.Context, subtestID int, testID int) (string, decimal.Decimal) {
	tf, err := os.CreateTemp("", "prog-out-*")
	if err != nil {
		return ErrOut, decimal.Zero
	}
	defer tf.Close()
	defer os.Remove(tf.Name())

	cf, err := os.CreateTemp("", "correct-out-*")
	if err != nil {
		return ErrOut, decimal.Zero
	}
	defer cf.Close()
	defer os.Remove(cf.Name())

	if err := redirBucketFile(tf, datastore.GetBucket(datastore.BucketTypeSubtests), strconv.Itoa(subtestID)); err != nil {
		return ErrOut, decimal.Zero
	}
	if err := redirBucketFile(cf, datastore.GetBucket(datastore.BucketTypeTests), strconv.Itoa(testID)+".out"); err != nil {
		return ErrOut, decimal.Zero
	}

	if err := exec.CommandContext(ctx, "diff", "-qBbEa", tf.Name(), cf.Name()).Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 0 {
				return CorrectOut, decimal.NewFromInt(100)
			}

			return WrongOut, decimal.Zero
		}

		return WrongOut, decimal.Zero
	}

	return CorrectOut, decimal.NewFromInt(100)
}

func redirBucketFile(w io.Writer, bucket *datastore.Bucket, filename string) error {
	f, err := bucket.Reader(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}
