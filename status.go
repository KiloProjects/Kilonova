package kilonova

import (
	"errors"
	"fmt"
	"log/slog"
)

var (
	ErrNoUpdates       = Statusf(400, "No updates specified")
	ErrMissingRequired = Statusf(400, "Missing required fields")

	ErrNotFound = Statusf(404, "Not found")

	ErrFeatureDisabled = Statusf(400, "Feature disabled by administrator")
)

var _ error = &statusError{}

type statusError struct {
	Code int
	Text string

	WrappedError error
}

func (s *statusError) LogValue() slog.Value {
	if s == nil {
		return slog.Value{}
	}
	return slog.StringValue(s.Text)
}

func (s *statusError) Error() string {
	return s.Text
}

func (s *statusError) Unwrap() error {
	return s.WrappedError
}

func (s *statusError) Is(target error) bool {
	if err, ok := target.(*statusError); ok {
		return err.Text == s.Text
	}
	return false
}

func Statusf(status int, format string, args ...any) error {
	return &statusError{Code: status, Text: fmt.Sprintf(format, args...)}
}

func ErrorCode(err error) int {
	if err == nil {
		return 200
	}
	var err2 *statusError
	if errors.As(err, &err2) {
		return err2.Code
	}
	return 500
}
