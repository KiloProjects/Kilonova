package kilonova

import (
	"embed"
	"errors"
	"fmt"
)

const Version = "v0.9.2"

//go:embed docs
var Docs embed.FS

// Error handling

const (
	EINTERNAL       = "internal"
	EINVALID        = "invalid"
	ENOTFOUND       = "not_found"
	ENOTIMPLEMENTED = "not_implemented"
	EUNAUTHORIZED   = "unauthorized"
)

// Errors that may be returned
var (
	ErrNoUpdates       = &Error{Code: EINVALID, Message: "No updates specified"}
	ErrMissingRequired = &Error{Code: EINVALID, Message: "Missing required fields"}
)

type Error struct {
	// Error code
	Code string

	// Human readable error message
	Message string

	// For Unwrap()
	Err error
}

func (e *Error) Error() string {
	if e.Err == nil || (e.Err != nil && e.Message == e.Err.Error()) {
		return fmt.Sprintf("Kilonova Error: code=%s message=%q", e.Code, e.Message)
	}
	return fmt.Sprintf("Kilonova Error: code=%s message=%q wrapped=%q", e.Code, e.Message, e.Err)
}

func (e *Error) Is(target error) bool {
	if err, ok := target.(*Error); ok {
		return e.Code == err.Code && e.Message == err.Message
	}
	return false
}

func (e *Error) Unwrap() error { return e.Err }

func FromError(code string, err error) error {
	return &Error{Code: code, Message: err.Error(), Err: err}
}

func WrapError(code, message string, err error) error {
	return &Error{Code: code, Message: message, Err: err}
}

func WrapInternal(err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: EINTERNAL, Err: err}
}

func ErrorCode(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Code
	}
	return EINTERNAL
}

func ErrorMessage(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Message
	}
	return "Internal error"
}

func Errorf(code string, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
