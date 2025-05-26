package media

import (
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
)

// CreateSeriesCommand represents a command to create a new series
type CreateSeriesCommand struct {
	Title        string
	Description  string
	FirstAirDate time.Time
	Genres       []string
	Networks     []string
}

// AddEpisodeCommand represents a command to add an episode to a series
type AddEpisodeCommand struct {
	SeriesID      uuid.UUID
	Title         string
	Description   string
	SeasonNumber  int
	EpisodeNumber int
	AirDate       time.Time
}

// CreateMovieCommand represents a command to create a new movie
type CreateMovieCommand struct {
	Title       string
	Description string
	ReleaseDate time.Time
	Genres      []string
	Director    string
	Cast        []string
}

// UpdateMediaStatusCommand represents a command to update media status
type UpdateMediaStatusCommand struct {
	MediaID   uuid.UUID
	MediaType string
	NewStatus media.Status
	FilePath  string // Optional, used when status is NeedsTranscode
}

// DeleteSeriesCommand represents a command to delete a series
type DeleteSeriesCommand struct {
	SeriesID uuid.UUID
}

// DeleteMovieCommand represents a command to delete a movie
type DeleteMovieCommand struct {
	MovieID uuid.UUID
}

// RemoveEpisodeCommand represents a command to remove an episode from a series
type RemoveEpisodeCommand struct {
	SeriesID  uuid.UUID
	EpisodeID uuid.UUID
}

// UpdateWatchProgressCommand represents a command to update watch progress
type UpdateWatchProgressCommand struct {
	MediaID      uuid.UUID
	UserID       uuid.UUID
	Position     time.Duration
	Completed    bool
	LastWatched  time.Time
}