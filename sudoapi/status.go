package sudoapi

import (
	"fmt"

	"github.com/KiloProjects/kilonova"
)

var (
	ErrNotFound     = kilonova.ErrNotFound
	ErrUnknownError = fmt.Errorf("unknown error occured")
)

// Reimplement Statusf and WrapError functions here for faster reference

func Statusf(status int, format string, args ...any) error {
	return kilonova.Statusf(status, format, args...)
}
