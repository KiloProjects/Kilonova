package datastore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// TODO: Has ... Get may not be the best pattern
func (m *StorageManager) HasAttachmentRender(attID int, renderType string) bool {
	m.attMu.RLock()
	defer m.attMu.RUnlock()
	if _, err := os.Stat(m.AttachmentRenderPath(attID, renderType)); err != nil {
		return false
	}
	return true
}

func (m *StorageManager) GetAttachmentRender(attID int, renderType string) (io.ReadCloser, error) {
	m.attMu.RLock()
	defer m.attMu.RUnlock()
	return os.Open(m.AttachmentRenderPath(attID, renderType))
}

func (m *StorageManager) DelAttachmentRender(attID int, renderType string) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	err := os.Remove(m.AttachmentRenderPath(attID, renderType))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (m *StorageManager) DelAttachmentRenders(attID int) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	entries, err := os.ReadDir(path.Join(m.RootPath, "attachments"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		parts := strings.SplitN(entry.Name(), ".", 2)
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			zap.S().Warn("Attachment renders should start with attachment ID:", entry.Name())
			continue
		}
		if id != attID {
			continue
		}
		if err := os.Remove(path.Join(m.RootPath, "attachments", entry.Name())); err != nil {
			zap.S().Warn("Could not delete attachment render: ", err)
			continue
		}
	}

	return nil
}

func (m *StorageManager) SaveAttachmentRender(attID int, renderType string, data []byte) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	return os.WriteFile(m.AttachmentRenderPath(attID, renderType), data, 0644)
}

func (m *StorageManager) InvalidateAllAttachments() error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	if err := os.RemoveAll(path.Join(m.RootPath, "attachments")); err != nil {
		return err
	}

	return os.MkdirAll(path.Join(m.RootPath, "attachments"), 0755)
}

func (m *StorageManager) AttachmentRenderPath(attID int, renderType string) string {
	return path.Join(m.RootPath, "attachments", strconv.Itoa(attID)+"."+renderType)
}
