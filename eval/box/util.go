package box

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova/eval"
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

func saveFile(p string, bucket eval.Bucket, filename string, mode fs.FileMode) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	return bucket.WriteFile(filename, f, mode)
}

func writeFile(p string, r io.Reader, mode fs.FileMode) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_SYNC, mode)
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
		slog.WarnContext(context.Background(), "File stat returned weird error", slog.String("path", p), slog.Any("err", err))
		return false
	}
	return true
}
