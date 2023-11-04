package datastore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
)

// TODO: Has ... Get may not be the best pattern
func (m *StorageManager) HasAttachmentRender(attID int) bool {
	m.attMu.RLock()
	defer m.attMu.RUnlock()
	if _, err := os.Stat(m.AttachmentRenderPath(attID)); err != nil {
		return false
	}
	return true
}

func (m *StorageManager) GetAttachmentRender(attID int) (io.ReadCloser, error) {
	m.attMu.RLock()
	defer m.attMu.RUnlock()
	return os.Open(m.AttachmentRenderPath(attID))
}

func (m *StorageManager) DelAttachmentRender(attID int) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	err := os.Remove(m.AttachmentRenderPath(attID))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (m *StorageManager) SaveAttachmentRender(attID int, data []byte) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	return os.WriteFile(m.AttachmentRenderPath(attID), data, 0644)
}

func (m *StorageManager) InvalidateAllAttachments() error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	if err := os.RemoveAll(path.Join(m.RootPath, "attachments")); err != nil {
		return err
	}

	return os.MkdirAll(path.Join(m.RootPath, "attachments"), 0777)
}

func (m *StorageManager) AttachmentRenderPath(attID int) string {
	return path.Join(m.RootPath, "attachments", strconv.Itoa(attID))
}
