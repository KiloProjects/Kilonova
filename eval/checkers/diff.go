package checkers

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/KiloProjects/kilonova/eval"
)

const (
	ErrOut     = "Internal checker error"
	CorrectOut = "Correct"
	WrongOut   = "Wrong Answer"
)

var _ eval.Checker = &DiffChecker{}

type DiffChecker struct{}

func (d *DiffChecker) Prepare(_ context.Context) (string, error) { return "", nil }

func (d *DiffChecker) Cleanup(_ context.Context) error { return nil }

func (d *DiffChecker) RunChecker(ctx context.Context, pOut, cOut io.Reader, maxScore int) (string, int) {
	tf, err := os.CreateTemp("", "prog-out-*")
	if err != nil {
		return ErrOut, 0
	}
	defer tf.Close()
	cf, err := os.CreateTemp("", "correct-out-*")
	if err != nil {
		return ErrOut, 0
	}
	defer cf.Close()

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
				return CorrectOut, maxScore
			}

			return WrongOut, 0
		}

		return WrongOut, 0
	}

	return CorrectOut, maxScore
}
