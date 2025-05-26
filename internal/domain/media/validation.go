package media

import (
	"path/filepath"
	"strings"
	"time"
)

// validateTitle validates a media title
func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return NewValidationError("title", "cannot be empty")
	}
	if len(title) > 255 {
		return NewValidationError("title", "cannot be longer than 255 characters")
	}
	return nil
}

// validateDescription validates a media description
func validateDescription(description string) error {
	if len(description) > 10000 {
		return NewValidationError("description", "cannot be longer than 10000 characters")
	}
	return nil
}

// validateSeasonNumber validates a season number
func validateSeasonNumber(seasonNumber int) error {
	if seasonNumber < 1 {
		return NewValidationError("seasonNumber", "must be greater than 0")
	}
	return nil
}

// validateEpisodeNumber validates an episode number
func validateEpisodeNumber(episodeNumber int) error {
	if episodeNumber < 1 {
		return NewValidationError("episodeNumber", "must be greater than 0")
	}
	return nil
}

// validateAirDate validates an air date
func validateAirDate(airDate time.Time) error {
	if airDate.IsZero() {
		return NewValidationError("airDate", "cannot be zero")
	}
	return nil
}

// validateReleaseDate validates a release date
func validateReleaseDate(releaseDate time.Time) error {
	if releaseDate.IsZero() {
		return NewValidationError("releaseDate", "cannot be zero")
	}
	return nil
}

// validateDuration validates a duration in minutes
func validateDuration(duration int) error {
	if duration < 1 {
		return NewValidationError("duration", "must be greater than 0")
	}
	if duration > 1000 {
		return NewValidationError("duration", "cannot be greater than 1000 minutes")
	}
	return nil
}

// validateFilePath validates a file path
func validateFilePath(filePath string) error {
	if strings.TrimSpace(filePath) == "" {
		return NewValidationError("filePath", "cannot be empty")
	}
	if !filepath.IsAbs(filePath) {
		return NewValidationError("filePath", "must be an absolute path")
	}
	return nil
}

// validateThumbnailPath validates a thumbnail path
func validateThumbnailPath(thumbnailPath string) error {
	if strings.TrimSpace(thumbnailPath) == "" {
		return NewValidationError("thumbnailPath", "cannot be empty")
	}
	if !filepath.IsAbs(thumbnailPath) {
		return NewValidationError("thumbnailPath", "must be an absolute path")
	}
	return nil
}

// validateStatus validates a media status
func validateStatus(status Status) error {
	switch status {
	case StatusPending, StatusTranscoding, StatusReady, StatusError:
		return nil
	default:
		return NewValidationError("status", "invalid status")
	}
} 