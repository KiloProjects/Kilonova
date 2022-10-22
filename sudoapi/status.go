package sudoapi

import (
	"github.com/KiloProjects/kilonova"
)

var (
	ErrNoUpdates       = kilonova.ErrNoUpdates
	ErrMissingRequired = kilonova.ErrMissingRequired

	ErrNotFound     = kilonova.ErrNotFound
	ErrUnknownError = kilonova.ErrUnknownError
)

type StatusError = kilonova.StatusError

// Reimplement Statusf and WrapError functions here for faster reference

func Statusf(status int, format string, args ...any) *StatusError {
	return kilonova.Statusf(status, format, args...)
}

func WrapError(err error, text string) *StatusError {
	return kilonova.WrapError(err, text)
}
