package eval

import (
	"io"
	"io/fs"
)

type Scratch interface {
	SaveFile(r io.Reader) (string, error)
	ReadFile(identifier string) (io.ReadCloser, error)
	DeleteFile(identifier string) error
}

type ScratchFile struct {
	Identifier string
	BoxPath    string
	Mode       fs.FileMode
}

type Box3Request struct {
	InputFiles []ScratchFile

	Command   []string
	RunConfig *RunConfig

	OutputFilePaths []string
}

type Multibox3Request struct {
	ManagerSandbox *Box3Request

	// OutputByteFiles/OutputBucketFiles are ignored.
	UserSandboxConfigs []*Box3Request

	// UseStdin is true if the user sandboxes read from stdin and write to stdout.
	// Otherwise, user processes read from and write to fifos whose paths are given as extra arguments.
	UseStdin bool
}

type Box3Response struct {
	Stats *RunStats

	// Files maps the output file path to a scratch identifier
	Files map[string]string
}
