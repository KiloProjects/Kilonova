package checkers

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/KiloProjects/kilonova/eval"
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

func (d *DiffChecker) RunChecker(ctx context.Context, pOut, cIn, cOut io.Reader) (string, int) {
	tf, err := os.CreateTemp("", "prog-out-*")
	if err != nil {
		return ErrOut, 0
	}
	defer tf.Close()
	defer os.Remove(tf.Name())

	cf, err := os.CreateTemp("", "correct-out-*")
	if err != nil {
		return ErrOut, 0
	}
	defer cf.Close()
	defer os.Remove(cf.Name())

	if _, err := io.Copy(tf, pOut); err != nil {
		return ErrOut, 0
	}
	if _, err := io.Copy(cf, cOut); err != nil {
		return ErrOut, 0
	}

	cmd := exec.CommandContext(ctx, "diff", "-qBbEa", tf.Name(), cf.Name())
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if err.ExitCode() == 0 {
				return CorrectOut, 100
			}

			return WrongOut, 0
		}

		return WrongOut, 0
	}

	return CorrectOut, 100
}
