package media

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"
	
	"github.com/google/uuid"
)

var (
	ErrInvalidPath     = errors.New("invalid file path")
)

// WatchProgress represents a user's watch progress for a media item
type WatchProgress struct {
	MediaID      uuid.UUID
	UserID       uuid.UUID
	Position     time.Duration
	Duration     time.Duration
	LastWatched  time.Time
	IsCompleted  bool
}

// Duration represents a media duration in seconds
type Duration struct {
	seconds int
}

// NewDuration creates a new Duration value object
func NewDuration(seconds int) (Duration, error) {
	if seconds <= 0 {
		return Duration{}, ErrInvalidDuration
	}
	return Duration{seconds: seconds}, nil
}

// Seconds returns the duration in seconds
func (d Duration) Seconds() int {
	return d.seconds
}

// String returns the duration in HH:MM:SS format
func (d Duration) String() string {
	duration := time.Duration(d.seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// FilePath represents a validated file path
type FilePath struct {
	path string
}

// NewFilePath creates a new FilePath value object
func NewFilePath(path string) (FilePath, error) {
	if path == "" {
		return FilePath{}, ErrInvalidPath
	}
	
	// Validate path format
	if !filepath.IsAbs(path) {
		return FilePath{}, fmt.Errorf("%w: path must be absolute", ErrInvalidPath)
	}
	
	return FilePath{path: path}, nil
}

// String returns the file path as a string
func (f FilePath) String() string {
	return f.path
}

// Metadata represents additional metadata for media items
type Metadata struct {
	Genres   []string
	Director string
	Cast     []string
	Rating   float32
	Language string
}

// NewMetadata creates a new Metadata value object
func NewMetadata(genres []string, director string, cast []string, rating float32, language string) Metadata {
	return Metadata{
		Genres:   genres,
		Director: director,
		Cast:     cast,
		Rating:   rating,
		Language: language,
	}
}

// HasGenre checks if the metadata contains a specific genre
func (m Metadata) HasGenre(genre string) bool {
	for _, g := range m.Genres {
		if g == genre {
			return true
		}
	}
	return false
}

// HasCastMember checks if the metadata contains a specific cast member
func (m Metadata) HasCastMember(actor string) bool {
	for _, a := range m.Cast {
		if a == actor {
			return true
		}
	}
	return false
} 