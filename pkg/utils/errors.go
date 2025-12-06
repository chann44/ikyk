package utils

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotFound         = errors.New("not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrInternalError    = errors.New("internal server error")
	ErrServiceUnavailable = errors.New("service unavailable")
)

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// NewError creates a new error with a formatted message
func NewError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
