package kilonova

import (
	"io"
	"time"
)

var (
	ErrDirectory    = &Error{Code: EINVALID, Message: "File is actually directory"}
	ErrNotDirectory = &Error{Code: EINVALID, Message: "Not a directory"}
	ErrNotEmpty     = &Error{Code: EINVALID, Message: "Directory you are trying to delete is not empty"}
	ErrNoDirInPath  = &Error{Code: EINVALID, Message: "Trying to save in a directory which is actually a file"}
	ErrNotExist     = &Error{Code: ENOTFOUND, Message: "Error doesn't exist"}
)

type TestStore interface {
	SaveTestInput(testID int, input io.Reader) error
	SaveTestOutput(testID int, output io.Reader) error
	TestInput(testID int) (io.ReadCloser, error)
	TestOutput(testID int) (io.ReadCloser, error)
}

type SubtestStore interface {
	SubtestWriter(subtest int) (io.WriteCloser, error)
	SubtestReader(subtest int) (io.ReadCloser, error)

	RemoveSubtestData(subtest int) error
}

// For CDN
type CDNStore interface {
	SaveFile(path string, r io.Reader) error
	CreateDir(path string) error
	// GetFile returns a ReadSeeker that must be Closed, the modtime and an error if anything occurs
	GetFile(path string) (io.ReadSeekCloser, time.Time, error)
	DeleteObject(path string) error
	ReadDir(path string) ([]CDNDirEntry, error)
}

// DataStore represents an interface for the Data Storage Manager
type DataStore interface {
	TestStore
	SubtestStore
	CDNStore
}

type CDNDirEntry struct {
	Type    string    `json:"type"`
	Name    string    `json:"name"`
	ModTime time.Time `json:"mod_time"`
	Size    int       `json:"size"`
}
