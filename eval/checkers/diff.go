package checkers

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/shopspring/decimal"
)

const (
	ErrOut     = "translate:internal_error"
	CorrectOut = "translate:success"
	WrongOut   = "translate:wrong"
)

var _ eval.Checker = &DiffChecker{}

type DiffChecker struct{}

func (d *DiffChecker) Prepare(_ context.Context) (string, error) { return "", nil }

func (d *DiffChecker) Cleanup(_ context.Context) error { return nil }

func (d *DiffChecker) RunChecker(ctx context.Context, pOut, cIn, cOut io.Reader) (string, decimal.Decimal) {
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

	if _, err := io.Copy(tf, pOut); err != nil {
		return ErrOut, decimal.Zero
	}
	if _, err := io.Copy(cf, cOut); err != nil {
		return ErrOut, decimal.Zero
	}

	cmd := exec.CommandContext(ctx, "diff", "-qBbEa", tf.Name(), cf.Name())
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if err.ExitCode() == 0 {
				return CorrectOut, decimal.NewFromInt(100)
			}

			return WrongOut, decimal.Zero
		}

		return WrongOut, decimal.Zero
	}

	return CorrectOut, decimal.NewFromInt(100)
}
