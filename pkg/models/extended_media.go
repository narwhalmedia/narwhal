package models

import (
	"time"

	"github.com/google/uuid"
)

// ExtendedMedia represents a more detailed media item structure
// This extends the basic Media type with additional fields for the GORM implementation
type ExtendedMedia struct {
	Media
	LibraryID      uuid.UUID  `json:"library_id"`
	OriginalTitle  string     `json:"original_title,omitempty"`
	Status         string     `json:"status"` // pending, available, missing, error
	FilePath       string     `json:"file_path"`
	FileSize       int64      `json:"file_size"`
	FileModifiedAt *time.Time `json:"file_modified_at,omitempty"`
	Description    string     `json:"description,omitempty"`
	ReleaseDate    time.Time  `json:"release_date,omitempty"`
	Runtime        int        `json:"runtime,omitempty"` // minutes
	Genres         []string   `json:"genres,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	TMDBID         int        `json:"tmdb_id,omitempty"`
	IMDBID         string     `json:"imdb_id,omitempty"`
	TVDBID         int        `json:"tvdb_id,omitempty"`
	MusicBrainzID  *uuid.UUID `json:"musicbrainz_id,omitempty"`
	VideoCodec     string     `json:"video_codec,omitempty"`
	AudioCodec     string     `json:"audio_codec,omitempty"`
	PosterURL      string     `json:"poster_url,omitempty"`
	BackdropURL    string     `json:"backdrop_url,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ExtendedEpisode represents a more detailed episode structure
type ExtendedEpisode struct {
	Episode
	Description string    `json:"description,omitempty"`
	Runtime     int       `json:"runtime,omitempty"` // minutes
	FileSize    int64     `json:"file_size,omitempty"`
	Status      string    `json:"status"` // available, missing, etc.
	VideoCodec  string    `json:"video_codec,omitempty"`
	AudioCodec  string    `json:"audio_codec,omitempty"`
	Resolution  string    `json:"resolution,omitempty"`
	Bitrate     int       `json:"bitrate,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}