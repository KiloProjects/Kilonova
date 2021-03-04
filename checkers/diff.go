package checkers

import (
	"io"
	"os"
	"os/exec"
)

type DiffChecker struct{}

func (d *DiffChecker) RunChecker(pOut io.Reader, cOut io.Reader, maxScore int) (string, int) {
	tf, err := os.CreateTemp("", "prog-out-*")
	if err != nil {
		return "Can't save program output", 0
	}
	defer tf.Close()
	cf, err := os.CreateTemp("", "correct-out-*")
	if err != nil {
		return "Can't save correct output", 0
	}
	defer cf.Close()

	cmd := exec.Command("diff", "-qBbEa", tf.Name(), cf.Name())
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if err.ExitCode() == 0 {
				return "Correct", maxScore
			}

			return "Wrong Answer", 0
		}

		return "Wrong Answer", 0
	}

	return "Correct", maxScore
}
