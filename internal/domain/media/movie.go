package media

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidMovieTitle = errors.New("movie title cannot be empty")
	ErrInvalidDirector   = errors.New("director cannot be empty")
	ErrInvalidDuration   = errors.New("duration must be positive")
)

// Movie represents a movie aggregate
type Movie struct {
	BaseAggregate
	title         string        // Private field to enforce encapsulation
	description   string
	releaseDate   time.Time
	genres        []string
	director      string
	cast          []string
	duration      time.Duration // Movie duration
	status        Status
	filePath      string
	thumbnailPath string
	rating        float32 // Rating out of 10
}

// NewMovie creates a new Movie aggregate with validation
func NewMovie(title, description string, releaseDate time.Time, genres []string, director string, cast []string) *Movie {
	movie := &Movie{
		BaseAggregate: NewBaseAggregate(),
		title:         title,
		description:   description,
		releaseDate:   releaseDate,
		genres:        genres,
		director:      director,
		cast:          cast,
		status:        StatusPending,
	}
	return movie
}

// Validate validates the movie aggregate
func (m *Movie) Validate() error {
	if m.title == "" {
		return ErrInvalidMovieTitle
	}
	if m.director == "" {
		return ErrInvalidDirector
	}
	if m.duration < 0 {
		return ErrInvalidDuration
	}
	return nil
}

// Title returns the movie title
func (m *Movie) Title() string {
	return m.title
}

// Description returns the movie description
func (m *Movie) Description() string {
	return m.description
}

// ReleaseDate returns the release date
func (m *Movie) ReleaseDate() time.Time {
	return m.releaseDate
}

// Genres returns a copy of genres
func (m *Movie) Genres() []string {
	genresCopy := make([]string, len(m.genres))
	copy(genresCopy, m.genres)
	return genresCopy
}

// Director returns the director
func (m *Movie) Director() string {
	return m.director
}

// Cast returns a copy of cast
func (m *Movie) Cast() []string {
	castCopy := make([]string, len(m.cast))
	copy(castCopy, m.cast)
	return castCopy
}

// Duration returns the movie duration
func (m *Movie) Duration() time.Duration {
	return m.duration
}

// SetDuration sets the movie duration
func (m *Movie) SetDuration(duration time.Duration) error {
	if duration < 0 {
		return ErrInvalidDuration
	}
	m.duration = duration
	m.incrementVersion()
	return nil
}

// Rating returns the movie rating
func (m *Movie) Rating() float32 {
	return m.rating
}

// SetRating sets the movie rating
func (m *Movie) SetRating(rating float32) error {
	if rating < 0 || rating > 10 {
		return errors.New("rating must be between 0 and 10")
	}
	m.rating = rating
	m.incrementVersion()
	return nil
}

// UpdateMetadata updates movie metadata
func (m *Movie) UpdateMetadata(title, description string, releaseDate time.Time, genres []string, director string, cast []string) error {
	if title == "" {
		return ErrInvalidMovieTitle
	}
	if director == "" {
		return ErrInvalidDirector
	}
	
	m.title = title
	m.description = description
	m.releaseDate = releaseDate
	m.genres = genres
	m.director = director
	m.cast = cast
	m.incrementVersion()
	
	return nil
}

// SetFilePath sets the file path for the movie
func (m *Movie) SetFilePath(filePath string) {
	m.filePath = filePath
	m.incrementVersion()
}

// GetFilePath returns the file path
func (m *Movie) GetFilePath() string {
	return m.filePath
}

// SetThumbnailPath sets the thumbnail path
func (m *Movie) SetThumbnailPath(thumbnailPath string) {
	m.thumbnailPath = thumbnailPath
	m.incrementVersion()
}

// GetThumbnailPath returns the thumbnail path
func (m *Movie) GetThumbnailPath() string {
	return m.thumbnailPath
}

// SetStatus updates the movie status
func (m *Movie) SetStatus(status Status) {
	m.status = status
	m.incrementVersion()
}

// GetStatus returns the movie status
func (m *Movie) GetStatus() Status {
	return m.status
}

// incrementVersion increments the version and updates timestamp
func (m *Movie) incrementVersion() {
	m.Version++
	m.UpdatedAt = time.Now()
}

// IsReadyToStream checks if the movie is ready for streaming
func (m *Movie) IsReadyToStream() bool {
	return m.status == StatusReady && m.filePath != ""
}

// NeedsTranscoding checks if the movie needs transcoding
func (m *Movie) NeedsTranscoding() bool {
	return m.status == StatusNeedsTranscode || (m.status == StatusPending && m.filePath != "")
}

// HasGenre checks if the movie has a specific genre
func (m *Movie) HasGenre(genre string) bool {
	for _, g := range m.genres {
		if g == genre {
			return true
		}
	}
	return false
}

// HasCastMember checks if a specific actor is in the cast
func (m *Movie) HasCastMember(actor string) bool {
	for _, member := range m.cast {
		if member == actor {
			return true
		}
	}
	return false
}

// GetYear returns the release year
func (m *Movie) GetYear() int {
	return m.releaseDate.Year()
}