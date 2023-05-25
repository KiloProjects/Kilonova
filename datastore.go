package kilonova

import (
	"io"
)

var (
	ErrDirectory    = Statusf(400, "File is actually directory")
	ErrNotDirectory = Statusf(400, "Not a directory")
	ErrNotEmpty     = Statusf(400, "Directory you are trying to delete is not empty")
	ErrNoDirInPath  = Statusf(400, "Trying to save in a directory which is actually a file")
	ErrNotExist     = Statusf(404, "Error doesn't exist")
)

type GraderStore interface {
	TestInput(testID int) (io.ReadSeekCloser, error)
	TestOutput(testID int) (io.ReadSeekCloser, error)

	SaveTestInput(testID int, input io.Reader) error
	SaveTestOutput(testID int, output io.Reader) error

	SubtestWriter(subtest int) (io.WriteCloser, error)
	SubtestReader(subtest int) (io.ReadSeekCloser, error)
	//GetDB(name string) (*sqlx.DB, error)
}

// DataStore represents an interface for the Data Storage Manager
type DataStore interface {
	GraderStore

	HasAttachmentRender(attID int) bool
	GetAttachmentRender(attID int) (io.ReadSeekCloser, error)
	DelAttachmentRender(attID int) error
	SaveAttachmentRender(attID int, data []byte) error

	InvalidateAllAttachments() error
}
