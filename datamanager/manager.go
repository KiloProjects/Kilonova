package datamanager

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/KiloProjects/Kilonova/common"
)

// StorageManager helps open the files in the data directory, this is supposed to be data that should not be stored in the DB
type StorageManager struct {
	RootPath string
}

// Manager represents an interface for the manager
type Manager interface {
	GetTest(pbID uint, testID uint) ([]byte, []byte, error)
	SaveTest(pbID, testID uint, input, output []byte) error

	GetAttachment(dir string, ID int, name string) ([]byte, error)
	SaveAttachment(dir string, ID int, data []byte, name string) error
}

// Session holds the session data of a specified user
type Session struct {
	User    common.User
	Expires time.Time
}

// GetTest returns a test for the specified problem
func (m *StorageManager) GetTest(pbID uint, testID uint) ([]byte, []byte, error) {
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
func (m *StorageManager) SaveTest(pbID, testID uint, input, output []byte) error {
	problem := strconv.FormatUint(uint64(pbID), 10)
	test := strconv.FormatUint(uint64(testID), 10)
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

// GetAttachment returns an attachment from disk
func (m *StorageManager) GetAttachment(dir string, ID int, name string) ([]byte, error) {
	return ioutil.ReadFile(path.Join(m.RootPath, "attachments", dir, strconv.Itoa(ID), name))
}

// SaveAttachment saves an attachment (ex: image) of something (specified with dir, right now the only thing you are should do is "problem") to disk
func (m *StorageManager) SaveAttachment(dir string, ID int, data []byte, name string) error {
	return ioutil.WriteFile(path.Join(m.RootPath, "attachments", dir, strconv.Itoa(ID), name), data, 0777)
}

// GetSession returns a session based on an ID
func (m *StorageManager) GetSession(id string) Session {
	// TODO
	return Session{}
}

// AddSession adds a session
func (m *StorageManager) AddSession(session Session) string {
	// TODO
	return ""
}

// NewManager returns a new manager instance
func NewManager(path string) *StorageManager {
	os.MkdirAll(path, 0777)
	return &StorageManager{RootPath: path}
}
