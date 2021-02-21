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

type CDNDirEntry struct {
	Type    string    `json:"type"`
	Name    string    `json:"name"`
	ModTime time.Time `json:"mod_time"`
	Size    int       `json:"size"`
}

// DataStore represents an interface for the Data Storage Manager
type DataStore interface {
	SaveTestInput(testID int, input io.Reader) error
	SaveTestOutput(testID int, output io.Reader) error
	TestInput(testID int) (io.ReadCloser, error)
	TestOutput(testID int) (io.ReadCloser, error)

	// I should merge SubtestWriter/SubtestReader sometime (returning an io.ReadWriteCloser), idk
	SubtestWriter(subtest int) (io.WriteCloser, error)
	SubtestReader(subtest int) (io.ReadCloser, error)

	RemoveSubtestData(subtest int) error

	// SubtestPath and TestOutputPath are bad workarounds for getting a valid path to the checker in internal/grader/grader.go.
	// I should fix this sometime, so i can have multiple different data store sources.
	SubtestPath(subtest int) string
	TestOutputPath(testID int) string

	// For CDN
	SaveFile(path string, r io.Reader) error
	CreateDir(path string) error
	// GetFile returns a ReadSeeker that must be Closed, the modtime and an error if anything occurs
	GetFile(path string) (io.ReadSeekCloser, time.Time, error)
	DeleteObject(path string) error
	ReadDir(path string) ([]CDNDirEntry, error)
}
