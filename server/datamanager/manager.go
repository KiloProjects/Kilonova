package datamanager

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

// Manager helps open the files in the data directory, this is supposed to be data that should not be stored in the database because it's a binary blob
type Manager struct {
	RootPath string
}

// GetTest returns a test buffer for the specified problem
func (m *Manager) GetTest(pbID int, testID int) (string, string, error) {
	input, err := ioutil.ReadFile(path.Join(m.RootPath, strconv.Itoa(pbID), "input", strconv.Itoa(testID)+".txt"))
	if err != nil {
		return "", "", err
	}
	output, err := ioutil.ReadFile(path.Join(m.RootPath, strconv.Itoa(pbID), "output", strconv.Itoa(testID)+".txt"))
	if err != nil {
		return "", "", err
	}
	return string(input), string(output), err
}

// NewManager returns a new manager instance
func NewManager(path string) *Manager {
	os.MkdirAll(path, 0777)
	return &Manager{RootPath: path}
}
