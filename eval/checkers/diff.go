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

type DiffChecker struct{ Store *datastore.Manager }

func (d *DiffChecker) Prepare(_ context.Context) (string, error) { return "", nil }

func (d *DiffChecker) Cleanup(_ context.Context) error { return nil }

func (d *DiffChecker) RunChecker(ctx context.Context, subtestID int, testID int) (string, decimal.Decimal) {
	tf, err := os.CreateTemp("", "prog-out-*")
	if err != nil {
		return ErrOut, decimal.Zero
	}
	defer os.Remove(tf.Name())
	defer tf.Close()

	cf, err := os.CreateTemp("", "correct-out-*")
	if err != nil {
		return ErrOut, decimal.Zero
	}
	defer os.Remove(cf.Name())
	defer cf.Close()

	if err := redirBucketFile(tf, d.Store.Subtests(), strconv.Itoa(subtestID)); err != nil {
		return ErrOut, decimal.Zero
	}
	if err := redirBucketFile(cf, d.Store.Tests(), strconv.Itoa(testID)+".out"); err != nil {
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

func redirBucketFile(w io.Writer, bucket datastore.Bucket, filename string) error {
	f, err := bucket.Reader(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}
