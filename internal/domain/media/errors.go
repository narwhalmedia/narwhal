package media

import (
	"errors"
	"fmt"
)

var (
	// ErrSeriesNotFound is returned when a series cannot be found
	ErrSeriesNotFound = errors.New("series not found")

	// ErrMovieNotFound is returned when a movie cannot be found
	ErrMovieNotFound = errors.New("movie not found")

	// ErrEpisodeNotFound is returned when an episode cannot be found
	ErrEpisodeNotFound = errors.New("episode not found")

	// ErrDuplicateSeries is returned when attempting to create a series with a duplicate title
	ErrDuplicateSeries = errors.New("series with this title already exists")
	
	// ErrSeriesAlreadyExists is an alias for ErrDuplicateSeries
	ErrSeriesAlreadyExists = ErrDuplicateSeries

	// ErrDuplicateMovie is returned when attempting to create a movie with a duplicate title
	ErrDuplicateMovie = errors.New("movie with this title already exists")
	
	// ErrMovieAlreadyExists is an alias for ErrDuplicateMovie
	ErrMovieAlreadyExists = ErrDuplicateMovie

	// ErrDuplicateEpisode is returned when attempting to add an episode with a duplicate season/episode number
	ErrDuplicateEpisode = errors.New("episode with this season and episode number already exists")

	// ErrInvalidStatus is returned when an invalid status is provided
	ErrInvalidStatus = errors.New("invalid status")

	// ErrInvalidDuration is returned when an invalid duration is provided
	ErrInvalidDuration = errors.New("invalid duration")

	// ErrInvalidFilePath is returned when an invalid file path is provided
	ErrInvalidFilePath = errors.New("invalid file path")

	// ErrInvalidThumbnailPath is returned when an invalid thumbnail path is provided
	ErrInvalidThumbnailPath = errors.New("invalid thumbnail path")
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
} 