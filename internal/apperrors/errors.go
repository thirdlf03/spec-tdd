package apperrors

import (
	"errors"
	"fmt"
)

// Common error types for the application
var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("not found")

	// ErrInvalidInput indicates invalid input was provided
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates an unauthorized access attempt
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInternal indicates an internal error occurred
	ErrInternal = errors.New("internal error")
)

// AppError represents an application-specific error with additional context
type AppError struct {
	Op      string // Operation that failed
	Err     error  // Underlying error
	Message string // User-friendly message
	Code    int    // Application-specific error code
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(op string, err error, message string) *AppError {
	return &AppError{
		Op:      op,
		Err:     err,
		Message: message,
	}
}

// Wrap wraps an error with an operation name
func Wrap(op string, err error) *AppError {
	return &AppError{
		Op:  op,
		Err: err,
	}
}

// Wrapf wraps an error with formatted context
func Wrapf(op string, err error, format string, args ...any) *AppError {
	return &AppError{
		Op:      op,
		Err:     err,
		Message: fmt.Sprintf(format, args...),
	}
}

// IsNotFound checks if an error is ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsInvalidInput checks if an error is ErrInvalidInput
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsUnauthorized checks if an error is ErrUnauthorized
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsInternal checks if an error is ErrInternal
func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}

// Join combines multiple errors into one
// This is a convenience wrapper around errors.Join from Go 1.20+
func Join(errs ...error) error {
	return errors.Join(errs...)
}
