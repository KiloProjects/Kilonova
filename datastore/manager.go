package datastore

import (
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
)

// StorageManager helps open the files in the data directory, this is supposed to be data that should not be stored in the DB
type StorageManager struct {
	RootPath string
}

type DBStorageManager struct {
	db *db.DB
}

var _ kilonova.DataStore = &StorageManager{}

//var _ kilonova.DataStore = &DBStorageManager{}

//func NewDBManager(db *db.DB) (kilonova.DataStore, error) {
//	return &DBStorageManager{p}, nil
//}

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

	return &StorageManager{p}, nil
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
