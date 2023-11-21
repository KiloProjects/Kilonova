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
	TestInput(testID int) (io.ReadCloser, error)
	TestOutput(testID int) (io.ReadCloser, error)

	PurgeTestData(testID int) error

	SaveTestInput(testID int, input io.Reader) error
	SaveTestOutput(testID int, output io.Reader) error

	SubtestWriter(subtest int) (io.WriteCloser, error)
	SubtestReader(subtest int) (io.ReadCloser, error)
}

// DataStore represents an interface for the Data Storage Manager
type DataStore interface {
	GraderStore

	HasAttachmentRender(attID int, renderType string) bool
	GetAttachmentRender(attID int, renderType string) (io.ReadCloser, error)
	DelAttachmentRender(attID int, renderType string) error
	// Like DelAttachmentRender but removes all renderTypes indiscriminately
	DelAttachmentRenders(attID int) error
	SaveAttachmentRender(attID int, renderType string, data []byte) error

	InvalidateAllAttachments() error
}
