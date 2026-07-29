package apperr

import "fmt"
import "errors"

type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func New(code, message string) error {
	return &Error{Code: code, Message: message}
}

func Wrap(code, message string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Message: message, Err: err}
}

func Code(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return "internal_error"
}
