package datastore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"vimagination.zapto.org/dos2unix"
)

// StorageManager helps open the files in the data directory, this is supposed to be data that should not be stored in the DB
type StorageManager struct {
	subtestBucket *Bucket
	testBucket    *Bucket
	avatarsBucket *Bucket

	attMu     sync.RWMutex
	attBucket *Bucket
}

var _ kilonova.DataStore = &StorageManager{}

// NewManager returns a new manager instance
func NewManager(p string) (kilonova.DataStore, error) {
	if err := os.MkdirAll(p, 0755); err != nil {
		return nil, err
	}

	sb, err := NewBucket(p, "subtests", NoCompression, false)
	if err != nil {
		return nil, err
	}

	tb, err := NewBucket(p, "tests", DefaultCompression, false)
	if err != nil {
		return nil, err
	}

	attb, err := NewBucket(p, "attachments", NoCompression, true)
	if err != nil {
		return nil, err
	}

	avb, err := NewBucket(p, "avatars", NoCompression, true)
	if err != nil {
		return nil, err
	}

	return &StorageManager{subtestBucket: sb, testBucket: tb, avatarsBucket: avb, attBucket: attb}, nil
}

// SubtestWriter should be used by the eval server
func (m *StorageManager) SubtestWriter(subtest int) (io.WriteCloser, error) {
	return m.subtestBucket.Writer(strconv.Itoa(subtest), 0644)
}

// SubtestReader should be used by the grader
func (m *StorageManager) SubtestReader(subtest int) (io.ReadCloser, error) {
	return m.subtestBucket.Reader(strconv.Itoa(subtest))
}

func (m *StorageManager) TestInput(testID int) (io.ReadCloser, error) {
	return m.testBucket.Reader(strconv.Itoa(testID) + ".in")
}

func (m *StorageManager) SaveTestInput(testID int, input io.Reader) error {
	return m.testBucket.WriteFile(strconv.Itoa(testID)+".in", dos2unix.DOS2Unix(input), 0644)
}

func (m *StorageManager) TestOutput(testID int) (io.ReadCloser, error) {
	return m.testBucket.Reader(strconv.Itoa(testID) + ".out")
}

func (m *StorageManager) SaveTestOutput(testID int, output io.Reader) error {
	return m.testBucket.WriteFile(strconv.Itoa(testID)+".out", dos2unix.DOS2Unix(output), 0644)
}

func (m *StorageManager) SaveAvatar(email string, size int, r io.Reader) error {
	return m.avatarsBucket.WriteFile(m.avatarName(email, size), r, 0644)
}

func (m *StorageManager) GetAvatar(email string, size int, maxLastMod time.Time) (io.ReadSeekCloser, time.Time, bool, error) {
	f, err := m.avatarsBucket.ReadSeeker(m.avatarName(email, size))
	if err != nil {
		return nil, time.Unix(0, 0), false, err
	}
	stat, err := m.avatarsBucket.Stat(m.avatarName(email, size))
	if err != nil {
		f.Close()
		return nil, time.Unix(0, 0), false, err
	}
	if stat.ModTime().Before(maxLastMod) {
		return f, stat.ModTime(), false, nil
	}
	return f, stat.ModTime(), true, nil
}

func (m *StorageManager) PurgeTestData(testID int) error {
	return errors.Join(
		m.testBucket.RemoveFile(strconv.Itoa(testID)+".in"),
		m.testBucket.RemoveFile(strconv.Itoa(testID)+".out"),
	)
}

func (m *StorageManager) PurgeAvatarCache() error {
	return m.avatarsBucket.ResetCache()
}

func (m *StorageManager) avatarName(email string, size int) string {
	bSum := sha256.Sum256([]byte(email))
	return fmt.Sprintf("%s-%d.png", hex.EncodeToString(bSum[:]), size)
}
