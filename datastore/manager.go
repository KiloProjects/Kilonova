package datastore

import (
	"io"
	"io/fs"
	"os"
	"path"
	"sync"

	"github.com/KiloProjects/kilonova"
)

// StorageManager helps open the files in the data directory, this is supposed to be data that should not be stored in the DB
type StorageManager struct {
	RootPath string

	attMu sync.RWMutex
}

var _ kilonova.DataStore = &StorageManager{}

// NewManager returns a new manager instance
func NewManager(p string) (kilonova.DataStore, error) {
	if err := os.MkdirAll(p, 0777); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "subtests"), 0777); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "tests"), 0777); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "dbs"), 0777); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(path.Join(p, "attachments"), 0777); err != nil {
		return nil, err
	}

	mgr := &StorageManager{RootPath: p}
	mgr.initDos2Unix()

	return mgr, nil
}

func writeFile(path string, r io.Reader, perms fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
