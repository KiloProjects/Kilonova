package datastore

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/KiloProjects/kilonova"
)

const (
	dir  = "directory"
	file = "file"
)

func (m *StorageManager) getCDNPath(fpath string) string {
	return path.Join(m.RootPath, "cdn", fpath)
}

func (m *StorageManager) GetFile(fpath string) (_ io.ReadSeekCloser, t time.Time, _ error) {
	fpath = m.getCDNPath(fpath)
	stat, err := os.Stat(fpath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, t, kilonova.ErrNotExist
		}
		return nil, t, kilonova.WrapInternal(err)
	}
	if stat.IsDir() {
		return nil, t, kilonova.ErrDirectory
	}
	f, err := os.Open(fpath)
	if err != nil {
		log.Println("Unknown error during opening of file:", err)
		return nil, t, kilonova.WrapInternal(err)
	}
	return f, stat.ModTime(), err
}

func (m *StorageManager) SaveFile(fpath string, r io.Reader) error {
	fpath = m.getCDNPath(fpath)
	if err := m.CreateDir(path.Dir(fpath)); err != nil {
		return err
	}
	return writeFile(fpath, r, 0777)
}

func (m *StorageManager) DeleteObject(fpath string) error {
	fpath = m.getCDNPath(fpath)
	if err := os.Remove(fpath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return kilonova.ErrNotExist
		}
		if errors.Is(err, syscall.ENOTEMPTY) {
			return kilonova.ErrNotEmpty
		}
		return kilonova.WrapInternal(err)
	}
	return nil
}

func (m *StorageManager) CreateDir(fpath string) error {
	fpath = m.getCDNPath(fpath)
	if err := os.MkdirAll(fpath, 0777); err != nil {
		if errors.Is(err, syscall.ENOTDIR) {
			return kilonova.ErrNoDirInPath
		}
		log.Println("Trying to create dir, encountered unknown error:", err)
		return err
	}
	return nil
}

func (m *StorageManager) ReadDir(fpath string) ([]kilonova.CDNDirEntry, error) {
	fpath = m.getCDNPath(fpath)
	osEntries, err := os.ReadDir(fpath)
	if err != nil {
		if errors.Is(err, fs.ErrInvalid) {
			return nil, kilonova.ErrNotDirectory
		}
		return nil, kilonova.WrapInternal(err)
	}
	// declare entries this way to return empty slice in case directory is empty
	entries := []kilonova.CDNDirEntry{}
	for _, osEntry := range osEntries {
		osInfo, err := osEntry.Info()
		if err != nil {
			log.Println("Unknown error while reading directory, skipping an entry:", err)
			continue
		}

		var entry kilonova.CDNDirEntry
		if osInfo.IsDir() {
			entry.Type = dir
			entry.Size = -1
		} else {
			entry.Type = file
			entry.Size = int(osInfo.Size())
		}
		entry.Name = osInfo.Name()
		entry.ModTime = osInfo.ModTime()
		entries = append(entries, entry)
	}

	return entries, nil
}
