package datastore

import (
	"io"
	"os"
	"path"
	"strconv"
)

// SubtestWriter should be used by the eval server
func (m *StorageManager) SubtestWriter(subtest int) (io.WriteCloser, error) {
	f, err := os.OpenFile(m.SubtestPath(subtest)+".gz", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return &gzipFileWriter{f, newGzipWriter(f)}, nil
}

// SubtestReader should be used by the grader
func (m *StorageManager) SubtestReader(subtest int) (io.ReadCloser, error) {
	return openGzipOrNormal(m.SubtestPath(subtest))
}

func (m *StorageManager) SubtestPath(subtest int) string {
	return path.Join(m.RootPath, "subtests", strconv.Itoa(subtest))
}
