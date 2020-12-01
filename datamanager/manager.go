package datamanager

import (
	"io/ioutil"
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
	Test(pbID int64, testID int64) ([]byte, []byte, error)
	SaveTest(pbID int64, testID int64, input, output []byte) error

	Attachment(dir string, ID int64, name string) ([]byte, error)
	SaveAttachment(dir string, ID int64, data []byte, name string) error
}

var _ Manager = &StorageManager{}

// Session holds the session data of a specified user
type Session struct {
	User    db.User
	Expires time.Time
}

// Test returns a test for the specified problem
func (m *StorageManager) Test(pbID int64, testID int64) ([]byte, []byte, error) {
	problem := strconv.FormatUint(uint64(pbID), 10)
	test := strconv.FormatUint(uint64(testID), 10)
	input, err := ioutil.ReadFile(path.Join(m.RootPath, "problems", problem, "input", test+".txt"))
	if err != nil {
		return []byte{}, []byte{}, err
	}
	output, err := ioutil.ReadFile(path.Join(m.RootPath, "problems", problem, "output", test+".txt"))
	if err != nil {
		return []byte{}, []byte{}, err
	}
	return input, output, err
}

// SaveTest saves an (input, output) pair of strings to disk to be used later as tests
func (m *StorageManager) SaveTest(pbID int64, testID int64, input, output []byte) error {
	problem := strconv.FormatInt(pbID, 10)
	test := strconv.FormatInt(int64(testID), 10)
	if err := os.MkdirAll(path.Join(m.RootPath, "problems", problem, "input"), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(m.RootPath, "problems", problem, "output"), 0777); err != nil {
		return err
	}
	if err := ioutil.WriteFile(
		path.Join(m.RootPath, "problems", problem, "input", test+".txt"),
		input, 0777); err != nil {
		return err
	}
	if err := ioutil.WriteFile(
		path.Join(m.RootPath, "problems", problem, "output", test+".txt"),
		output, 0777); err != nil {
		return err
	}
	return nil
}

// (|Save)Attachment are considered deprecated until further notice

// Attachment returns an "attachment" from disk
func (m *StorageManager) Attachment(dir string, ID int64, name string) ([]byte, error) {
	return ioutil.ReadFile(path.Join(m.RootPath, "attachments", dir, strconv.FormatInt(ID, 10), name))
}

// SaveAttachment saves an "attachment"
func (m *StorageManager) SaveAttachment(dir string, ID int64, data []byte, name string) error {
	return ioutil.WriteFile(path.Join(m.RootPath, "attachments", dir, strconv.FormatInt(ID, 10), name), data, 0777)
}

// NewManager returns a new manager instance
func NewManager(path string) *StorageManager {
	os.MkdirAll(path, 0777)
	return &StorageManager{RootPath: path}
}
