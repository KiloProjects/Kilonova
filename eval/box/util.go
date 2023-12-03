package box

import (
	"errors"
	"io"
	"io/fs"
	"os"

	"go.uber.org/zap"
)

func readFile(p string, w io.Writer) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func writeFile(p string, r io.Reader, mode fs.FileMode) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err1 := f.Sync(); err1 != nil && err == nil {
		err = err1
	}
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

func checkFile(p string) bool {
	_, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
		zap.S().Warnf("File stat (%q) returned weird error: %s", p, err)
		return false
	}
	return true
}
