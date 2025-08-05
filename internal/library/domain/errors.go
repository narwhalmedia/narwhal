package domain

import "errors"

// Common domain errors
var (
	// ErrLibraryNotFound is returned when a library is not found
	ErrLibraryNotFound = errors.New("library not found")

	// ErrMediaNotFound is returned when a media item is not found
	ErrMediaNotFound = errors.New("media not found")

	// ErrScanInProgress is returned when trying to start a scan while one is already running
	ErrScanInProgress = errors.New("scan already in progress")

	// ErrInvalidPath is returned when a library path is invalid
	ErrInvalidPath = errors.New("invalid library path")

	// ErrDuplicatePath is returned when trying to create a library with an existing path
	ErrDuplicatePath = errors.New("library path already exists")

	// ErrInvalidMediaType is returned when an invalid media type is provided
	ErrInvalidMediaType = errors.New("invalid media type")
)
