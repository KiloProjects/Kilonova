package sudoapi

import (
	"github.com/KiloProjects/kilonova"
)

var (
	ErrNotFound     = kilonova.ErrNotFound
	ErrUnknownError = Statusf(500, "Unknown error occured")
)

// Reimplement Statusf and WrapError functions here for faster reference

func Statusf(status int, format string, args ...any) error {
	return kilonova.Statusf(status, format, args...)
}

func WrapError(err error, text string) error {
	return kilonova.WrapError(err, text)
}
