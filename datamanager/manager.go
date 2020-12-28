package datamanager

import (
	"fmt"
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
	TestInput(testID int64) (io.ReadCloser, error)
	TestOutput(testID int64) (io.ReadCloser, error)

	SaveTestInput(testID int64, input io.Reader) error
	SaveTestOutput(testID int64, output io.Reader) error

	SubtestWriter(subtest int64) (io.WriteCloser, error)
	SubtestReader(subtest int64) (io.ReadCloser, error)
}

var _ Manager = &StorageManager{}

// Session holds the session data of a specified user
type Session struct {
	User    db.User
	Expires time.Time
}

// OldTestInput is used just for the migration tool, TODO: Delete in v0.6.2
func (m *StorageManager) OldTestInput(pbID int64, testID int64) (io.ReadCloser, error) {
	return os.Open(path.Join(m.RootPath, "problems", strconv.FormatInt(pbID, 10), "input", strconv.FormatInt(testID, 10)+".txt"))
}

// OldTestOutput is used just for the migration tool, TODO: Delete in v0.6.2
func (m *StorageManager) OldTestOutput(pbID int64, testID int64) (io.ReadCloser, error) {
	return os.Open(path.Join(m.RootPath, "problems", strconv.FormatInt(pbID, 10), "output", strconv.FormatInt(testID, 10)+".txt"))
}

func (m *StorageManager) TestInput(testID int64) (io.ReadCloser, error) {
	return os.Open(path.Join(m.RootPath, "tests", strconv.FormatInt(testID, 10)+".in"))
}

func (m *StorageManager) TestOutput(testID int64) (io.ReadCloser, error) {
	return os.Open(path.Join(m.RootPath, "tests", strconv.FormatInt(testID, 10)+".out"))
}

func (m *StorageManager) SaveTestInput(testID int64, input io.Reader) error {
	return writeFile(path.Join(m.RootPath, "tests", fmt.Sprintf("%d.in", testID)), input, 0777)
}

func (m *StorageManager) SaveTestOutput(testID int64, output io.Reader) error {
	return writeFile(path.Join(m.RootPath, "tests", fmt.Sprintf("%d.out", testID)), output, 0777)
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

	if err := os.MkdirAll(path.Join(p, "tests"), 0777); err != nil {
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
