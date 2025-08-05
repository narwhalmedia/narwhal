package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeNotFound indicates a resource was not found
	ErrorTypeNotFound ErrorType = "NOT_FOUND"
	// ErrorTypeBadRequest indicates a bad request
	ErrorTypeBadRequest ErrorType = "BAD_REQUEST"
	// ErrorTypeConflict indicates a conflict
	ErrorTypeConflict ErrorType = "CONFLICT"
	// ErrorTypeUnauthorized indicates unauthorized access
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	// ErrorTypeForbidden indicates forbidden access
	ErrorTypeForbidden ErrorType = "FORBIDDEN"
	// ErrorTypeInternal indicates an internal error
	ErrorTypeInternal ErrorType = "INTERNAL"
)

// AppError represents an application error
type AppError struct {
	Type    ErrorType
	Message string
	Err     error
}

// Error returns the error message
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new application error
func New(errorType ErrorType, message string) error {
	return &AppError{
		Type:    errorType,
		Message: message,
	}
}

// Wrap wraps an error with an application error
func Wrap(errorType ErrorType, message string, err error) error {
	return &AppError{
		Type:    errorType,
		Message: message,
		Err:     err,
	}
}

// NotFound creates a not found error
func NotFound(message string) error {
	return New(ErrorTypeNotFound, message)
}

// BadRequest creates a bad request error
func BadRequest(message string) error {
	return New(ErrorTypeBadRequest, message)
}

// Conflict creates a conflict error
func Conflict(message string) error {
	return New(ErrorTypeConflict, message)
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) error {
	return New(ErrorTypeUnauthorized, message)
}

// Forbidden creates a forbidden error
func Forbidden(message string) error {
	return New(ErrorTypeForbidden, message)
}

// Internal creates an internal error
func Internal(message string) error {
	return New(ErrorTypeInternal, message)
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeNotFound
	}
	return false
}

// IsBadRequest checks if an error is a bad request error
func IsBadRequest(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeBadRequest
	}
	return false
}

// IsConflict checks if an error is a conflict error
func IsConflict(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeConflict
	}
	return false
}

// IsUnauthorized checks if an error is an unauthorized error
func IsUnauthorized(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeUnauthorized
	}
	return false
}

// IsForbidden checks if an error is a forbidden error
func IsForbidden(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeForbidden
	}
	return false
}

// IsInternal checks if an error is an internal error
func IsInternal(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == ErrorTypeInternal
	}
	return false
}

// IsDuplicateError checks if an error is a duplicate key error
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "UNIQUE constraint") ||
		strings.Contains(errStr, "duplicate entry")
}
