package kilonova

import (
	"io"
)

var (
	ErrDirectory    = &Error{Code: EINVALID, Message: "File is actually directory"}
	ErrNotDirectory = &Error{Code: EINVALID, Message: "Not a directory"}
	ErrNotEmpty     = &Error{Code: EINVALID, Message: "Directory you are trying to delete is not empty"}
	ErrNoDirInPath  = &Error{Code: EINVALID, Message: "Trying to save in a directory which is actually a file"}
	ErrNotExist     = &Error{Code: ENOTFOUND, Message: "Error doesn't exist"}
)

type GraderStore interface {
	TestInput(testID int) (io.ReadCloser, error)
	TestOutput(testID int) (io.ReadCloser, error)

	SaveTestInput(testID int, input io.Reader) error
	SaveTestOutput(testID int, output io.Reader) error

	SubtestWriter(subtest int) (io.WriteCloser, error)
	SubtestReader(subtest int) (io.ReadCloser, error)

	//GetDB(name string) (*sqlx.DB, error)
}

// DataStore represents an interface for the Data Storage Manager
type DataStore = GraderStore
