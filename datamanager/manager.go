package datamanager

import (
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/KiloProjects/Kilonova/internal/db"
)

// StorageManager helps open the files in the data directory, this is supposed to be data that should not be stored in the DB
type StorageManager struct {
	RootPath string
}

// Manager represents an interface for the manager
type Manager interface {
	TestInput(pbID int64, testID int64) (io.ReadCloser, error)
	TestOutput(pbID int64, testID int64) (io.ReadCloser, error)
	Test(pbID int64, testID int64) (io.ReadCloser, io.ReadCloser, error)

	SaveTest(pbID int64, testID int64, input, output io.Reader) error

	SubtestWriter(subtest int64) (io.WriteCloser, error)
	SubtestReader(subtest int64) (io.ReadCloser, error)
}

var _ Manager = &StorageManager{}

// Session holds the session data of a specified user
type Session struct {
	User    db.User
	Expires time.Time
}

func (m *StorageManager) TestInput(pbID int64, testID int64) (io.ReadCloser, error) {
	return os.Open(path.Join(m.RootPath, "problems", strconv.FormatInt(pbID, 10), "input", strconv.FormatInt(testID, 10)+".txt"))
}
func (m *StorageManager) TestOutput(pbID int64, testID int64) (io.ReadCloser, error) {
	return os.Open(path.Join(m.RootPath, "problems", strconv.FormatInt(pbID, 10), "output", strconv.FormatInt(testID, 10)+".txt"))
}

// Test returns a test for the specified problem
func (m *StorageManager) Test(pbID int64, testID int64) (io.ReadCloser, io.ReadCloser, error) {
	input, err := m.TestInput(pbID, testID)
	if err != nil {
		return nil, nil, err
	}
	output, err := m.TestOutput(pbID, testID)
	if err != nil {
		input.Close()
		return nil, nil, err
	}
	return input, output, err
}

// SaveTest saves an (input, output) pair of strings to disk to be used later as tests
func (m *StorageManager) SaveTest(pbID int64, testID int64, input, output io.Reader) error {
	problem := strconv.FormatInt(pbID, 10)
	test := strconv.FormatInt(testID, 10)
	if err := os.MkdirAll(path.Join(m.RootPath, "problems", problem, "input"), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(m.RootPath, "problems", problem, "output"), 0777); err != nil {
		return err
	}
	if err := writeFile(
		path.Join(m.RootPath, "problems", problem, "input", test+".txt"),
		input, 0777); err != nil {
		return err
	}
	if err := writeFile(
		path.Join(m.RootPath, "problems", problem, "output", test+".txt"),
		output, 0777); err != nil {
		return err
	}
	return nil
}

// SubtestWriter should be used by the eval server
func (m *StorageManager) SubtestWriter(subtest int64) (io.WriteCloser, error) {
	return os.OpenFile(m.subtestPath(subtest), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
}

// SubtestReader should be used by the grader
func (m *StorageManager) SubtestReader(subtest int64) (io.ReadCloser, error) {
	return os.Open(m.subtestPath(subtest))
}

// NewManager returns a new manager instance
func NewManager(p string) (*StorageManager, error) {
	if err := os.MkdirAll(p, 0777); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "subtests"), 0777); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "problems"), 0777); err != nil {
		return nil, err
	}

	return &StorageManager{RootPath: p}, nil
}

func (m *StorageManager) subtestPath(subtest int64) string {
	return path.Join(m.RootPath, "subtests", strconv.FormatInt(subtest, 10))
}

func writeFile(path string, r io.Reader, perms fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	_, err = f.ReadFrom(r)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
