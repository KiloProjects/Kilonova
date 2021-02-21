package datastore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"

	"github.com/davecgh/go-spew/spew"
)

// SubtestWriter should be used by the eval server
func (m *StorageManager) SubtestWriter(subtest int) (io.WriteCloser, error) {
	return os.OpenFile(m.SubtestPath(subtest), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
}

// SubtestReader should be used by the grader
func (m *StorageManager) SubtestReader(subtest int) (io.ReadCloser, error) {
	return os.Open(m.SubtestPath(subtest))
}

func (m *StorageManager) RemoveSubtestData(subtest int) error {
	err := os.Remove(m.SubtestPath(subtest))
	spew.Dump(err)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (m *StorageManager) SubtestPath(subtest int) string {
	return path.Join(m.RootPath, "subtests", strconv.Itoa(subtest))
}
