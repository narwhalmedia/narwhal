package media

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEpisodeNotFound    = errors.New("episode not found")
	ErrInvalidEpisodeTitle = errors.New("episode title cannot be empty")
	ErrInvalidEpisodeNumber = errors.New("episode number must be positive")
)

// Episode represents a TV show episode
type Episode struct {
	BaseAggregate
	seriesID      uuid.UUID     // Reference to parent series
	title         string        // Private field to enforce encapsulation
	description   string
	seasonNumber  int
	episodeNumber int
	airDate       time.Time
	duration      time.Duration
	status        Status
	filePath      string
	thumbnailPath string
}

// NewEpisode creates a new Episode
func NewEpisode(title, description string, seasonNumber, episodeNumber int, airDate time.Time) *Episode {
	episode := &Episode{
		BaseAggregate: NewBaseAggregate(),
		title:         title,
		description:   description,
		seasonNumber:  seasonNumber,
		episodeNumber: episodeNumber,
		airDate:       airDate,
		status:        StatusPending,
	}
	return episode
}

// Validate validates the episode
func (e *Episode) Validate() error {
	if e.title == "" {
		return ErrInvalidEpisodeTitle
	}
	if e.seasonNumber < 1 {
		return ErrInvalidSeasonNumber
	}
	if e.episodeNumber < 1 {
		return ErrInvalidEpisodeNumber
	}
	if e.duration < 0 {
		return ErrInvalidDuration
	}
	return nil
}

// SeriesID returns the series ID
func (e *Episode) SeriesID() uuid.UUID {
	return e.seriesID
}

// SetSeriesID sets the series ID (used when adding to series)
func (e *Episode) SetSeriesID(seriesID uuid.UUID) {
	e.seriesID = seriesID
	e.incrementVersion()
}

// Title returns the episode title
func (e *Episode) Title() string {
	return e.title
}

// Description returns the episode description
func (e *Episode) Description() string {
	return e.description
}

// SeasonNumber returns the season number
func (e *Episode) SeasonNumber() int {
	return e.seasonNumber
}

// EpisodeNumber returns the episode number
func (e *Episode) EpisodeNumber() int {
	return e.episodeNumber
}

// AirDate returns the air date
func (e *Episode) AirDate() time.Time {
	return e.airDate
}

// Duration returns the episode duration
func (e *Episode) Duration() time.Duration {
	return e.duration
}

// SetDuration sets the episode duration
func (e *Episode) SetDuration(duration time.Duration) error {
	if duration < 0 {
		return ErrInvalidDuration
	}
	e.duration = duration
	e.incrementVersion()
	return nil
}

// GetEpisodeCode returns the episode code (e.g., "S01E01")
func (e *Episode) GetEpisodeCode() string {
	return fmt.Sprintf("S%02dE%02d", e.seasonNumber, e.episodeNumber)
}

// UpdateMetadata updates episode metadata
func (e *Episode) UpdateMetadata(title, description string, airDate time.Time) error {
	if title == "" {
		return ErrInvalidEpisodeTitle
	}
	
	e.title = title
	e.description = description
	e.airDate = airDate
	e.incrementVersion()
	
	return nil
}

// SetFilePath sets the file path for the episode
func (e *Episode) SetFilePath(filePath string) {
	e.filePath = filePath
	e.incrementVersion()
}

// GetFilePath returns the file path
func (e *Episode) GetFilePath() string {
	return e.filePath
}

// SetThumbnailPath sets the thumbnail path
func (e *Episode) SetThumbnailPath(thumbnailPath string) {
	e.thumbnailPath = thumbnailPath
	e.incrementVersion()
}

// GetThumbnailPath returns the thumbnail path
func (e *Episode) GetThumbnailPath() string {
	return e.thumbnailPath
}

// SetStatus updates the episode status
func (e *Episode) SetStatus(status Status) {
	e.status = status
	e.incrementVersion()
}

// GetStatus returns the episode status
func (e *Episode) GetStatus() Status {
	return e.status
}

// incrementVersion increments the version and updates timestamp
func (e *Episode) incrementVersion() {
	e.Version++
	e.UpdatedAt = time.Now()
}

// IsReadyToStream checks if the episode is ready for streaming
func (e *Episode) IsReadyToStream() bool {
	return e.status == StatusReady && e.filePath != ""
}

// NeedsTranscoding checks if the episode needs transcoding
func (e *Episode) NeedsTranscoding() bool {
	return e.status == StatusNeedsTranscode || (e.status == StatusPending && e.filePath != "")
}

// IsAiredBefore checks if the episode aired before a given date
func (e *Episode) IsAiredBefore(date time.Time) bool {
	return e.airDate.Before(date)
}

// IsAiredAfter checks if the episode aired after a given date
func (e *Episode) IsAiredAfter(date time.Time) bool {
	return e.airDate.After(date)
}