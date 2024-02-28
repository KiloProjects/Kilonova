package datastore

import (
	"bytes"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// TODO: Has ... Get may not be the best pattern
func (m *StorageManager) HasAttachmentRender(attID int, renderType string) bool {
	m.attMu.RLock()
	defer m.attMu.RUnlock()
	if _, err := m.attBucket.Stat(m.AttachmentName(attID, renderType)); err != nil {
		return false
	}
	return true
}

func (m *StorageManager) GetAttachmentRender(attID int, renderType string) (io.ReadCloser, error) {
	m.attMu.RLock()
	defer m.attMu.RUnlock()
	return m.attBucket.Reader(m.AttachmentName(attID, renderType))
}

func (m *StorageManager) DelAttachmentRender(attID int, renderType string) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	return m.attBucket.RemoveFile(m.AttachmentName(attID, renderType))
}

func (m *StorageManager) DelAttachmentRenders(attID int) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	return m.attBucket.IterFiles(func(entry fs.DirEntry) error {
		prefix, _, _ := strings.Cut(entry.Name(), ".")
		id, err := strconv.Atoi(prefix)
		if err != nil {
			zap.S().Warn("Attachment renders should start with attachment ID:", entry.Name())
			return nil
		}
		if id != attID {
			return nil
		}
		if err := m.attBucket.RemoveFile(path.Join(m.RootPath, "attachments", entry.Name())); err != nil {
			zap.S().Warn("Could not delete attachment render: ", err)
		}
		return nil
	})
}

func (m *StorageManager) SaveAttachmentRender(attID int, renderType string, data []byte) error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	return m.attBucket.WriteFile(m.AttachmentName(attID, renderType), bytes.NewReader(data), 0644)
}

func (m *StorageManager) InvalidateAllAttachments() error {
	m.attMu.Lock()
	defer m.attMu.Unlock()
	return m.attBucket.ResetCache()
}

func (m *StorageManager) AttachmentName(attID int, renderType string) string {
	return path.Join(strconv.Itoa(attID) + "." + renderType)
}
