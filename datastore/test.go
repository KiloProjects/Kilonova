package datastore

import (
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
)

func (m *StorageManager) TestInput(testID int) (io.ReadSeekCloser, error) {
	return os.Open(path.Join(m.RootPath, "tests", strconv.Itoa(testID)+".in"))
}

func (m *StorageManager) TestOutput(testID int) (io.ReadSeekCloser, error) {
	return os.Open(m.TestOutputPath(testID))
}

func (m *StorageManager) TestOutputPath(testID int) string {
	return path.Join(m.RootPath, "tests", strconv.Itoa(testID)+".out")
}

func (m *StorageManager) SaveTestInput(testID int, input io.Reader) error {
	return writeFile(path.Join(m.RootPath, "tests", fmt.Sprintf("%d.in", testID)), input, 0777)
}

func (m *StorageManager) SaveTestOutput(testID int, output io.Reader) error {
	return writeFile(path.Join(m.RootPath, "tests", fmt.Sprintf("%d.out", testID)), output, 0777)
}
