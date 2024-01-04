package kilonova

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

var (
	ErrNoUpdates       = Statusf(400, "No updates specified")
	ErrMissingRequired = Statusf(400, "Missing required fields")

	ErrNotFound     = Statusf(404, "Not found")
	ErrUnknownError = Statusf(500, "Unknown error occured")

	ErrFeatureDisabled = Statusf(400, "Feature disabled by administrator")

	ErrContextCanceled = WrapError(context.Canceled, "Context canceled")
)

var _ error = &StatusError{}

type StatusError struct {
	Code int
	Text string

	WrappedError error
}

func (s *StatusError) Error() string {
	return s.String()
}

func (s *StatusError) String() string {
	if s == nil {
		return "<No error>"
	}
	return fmt.Sprintf("<%d %q>", s.Code, s.Text)
}

func (s *StatusError) WriteError(w http.ResponseWriter) {
	if s == nil {
		zap.S().Warn("Attempted to send nil *StatusError over http.ResponseWriter.")
		return
	}
	StatusData(w, "error", s.Text, s.Code)
}

func (s *StatusError) Unwrap() error {
	return s.WrappedError
}

func (s *StatusError) Is(target error) bool {
	if err, ok := target.(*StatusError); ok {
		return err.Code == s.Code && err.Text == s.Text
	}
	return false
}

func Statusf(status int, format string, args ...any) *StatusError {
	return &StatusError{Code: status, Text: fmt.Sprintf(format, args...)}
}

func WrapError(err error, text string) *StatusError {
	if err == nil {
		return &StatusError{Code: 500, Text: text}
	}
	if text != "" {
		text += ": "
	}
	if err1, ok := err.(*StatusError); ok {
		return &StatusError{Code: err1.Code, Text: text + err1.Text, WrappedError: err1}
	}
	return &StatusError{Code: 500, Text: text + err.Error(), WrappedError: err}
}
