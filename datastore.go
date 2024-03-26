package kilonova

import (
	"io/fs"
)

var (
	ErrNotExist = Statusf(404, "File doesn't exist")
)

// TODO: Bucket interface

func init() {
	ErrNotExist.WrappedError = fs.ErrNotExist
}
