package datastore

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"
)

func (m *StorageManager) avatarPath(email string, size int) string {
	bSum := md5.Sum([]byte(email))
	return path.Join(m.RootPath, "avatars", fmt.Sprintf("%s-%d.png", hex.EncodeToString(bSum[:]), size))
}

func (m *StorageManager) SaveAvatar(email string, size int, r io.Reader) error {
	return writeFile(m.avatarPath(email, size), r, 0644)
}

func (m *StorageManager) GetAvatar(email string, size int, maxLastMod time.Time) (io.ReadSeeker, time.Time, bool, error) {
	f, err := os.Open(m.avatarPath(email, size))
	if err != nil {
		return nil, time.Unix(0, 0), false, err
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, time.Unix(0, 0), false, err
	}
	if stat.ModTime().Before(maxLastMod) {
		return f, stat.ModTime(), false, nil
	}
	return f, stat.ModTime(), true, nil
}

func (m *StorageManager) PurgeAvatarCache() error {
	if err := os.RemoveAll(path.Join(m.RootPath, "avatars")); err != nil {
		return err
	}

	return os.MkdirAll(path.Join(m.RootPath, "avatars"), 0755)
}

func writeFile(p string, r io.Reader, mode fs.FileMode) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
