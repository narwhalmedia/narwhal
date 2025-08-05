package models

import (
	"time"

	"github.com/google/uuid"
)

// MediaType represents the type of media content
type MediaType string

const (
	MediaTypeMovie  MediaType = "movie"
	MediaTypeSeries MediaType = "series"
	MediaTypeTV     MediaType = "tv"      // Alias for series
	MediaTypeMusic  MediaType = "music"
)

// Media represents a media item in the library
type Media struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	LibraryID   uuid.UUID  `json:"library_id" db:"library_id"`
	Title       string     `json:"title" db:"title"`
	Type        MediaType  `json:"type" db:"type"`
	Path        string     `json:"path" db:"path"`
	Size        int64      `json:"size" db:"size"`
	Duration    int        `json:"duration" db:"duration"` // in seconds
	Resolution  string     `json:"resolution,omitempty" db:"resolution"`
	Codec       string     `json:"codec,omitempty" db:"codec"`
	Bitrate     int        `json:"bitrate,omitempty" db:"bitrate"`
	Added       time.Time  `json:"added" db:"added"`
	Modified    time.Time  `json:"modified" db:"modified"`
	LastScanned time.Time  `json:"last_scanned" db:"last_scanned"`
	Metadata    *Metadata  `json:"metadata,omitempty"`
	Episodes    []*Episode `json:"episodes,omitempty"` // For series
	
	// Extended fields for GORM compatibility
	Status         string     `json:"status,omitempty" db:"status"`
	FilePath       string     `json:"file_path,omitempty" db:"file_path"`
	FileSize       int64      `json:"file_size,omitempty" db:"file_size"`
	FileModifiedAt *time.Time `json:"file_modified_at,omitempty" db:"file_modified_at"`
	Description    string     `json:"description,omitempty" db:"description"`
	ReleaseDate    time.Time  `json:"release_date,omitempty" db:"release_date"`
	Genres         []string   `json:"genres,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	TMDBID         int        `json:"tmdb_id,omitempty" db:"tmdb_id"`
	IMDBID         string     `json:"imdb_id,omitempty" db:"imdb_id"`
	TVDBID         int        `json:"tvdb_id,omitempty" db:"tvdb_id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	Year           int        `json:"year,omitempty" db:"year"`
}

// Episode represents an episode of a series
type Episode struct {
	ID           uuid.UUID `json:"id" db:"id"`
	MediaID      uuid.UUID `json:"media_id" db:"media_id"`
	SeasonNumber int       `json:"season_number" db:"season_number"`
	EpisodeNumber int      `json:"episode_number" db:"episode_number"`
	Title        string    `json:"title" db:"title"`
	Path         string    `json:"path" db:"path"`
	Duration     int       `json:"duration" db:"duration"`
	AirDate      time.Time `json:"air_date,omitempty" db:"air_date"`
	Added        time.Time `json:"added" db:"added"`
}

// Metadata contains enriched metadata for media items
type Metadata struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	MediaID      uuid.UUID  `json:"media_id" db:"media_id"`
	Title        string     `json:"title,omitempty" db:"title"`
	IMDBID       string     `json:"imdb_id,omitempty" db:"imdb_id"`
	TMDBID       string     `json:"tmdb_id,omitempty" db:"tmdb_id"`
	TVDBID       string     `json:"tvdb_id,omitempty" db:"tvdb_id"`
	Description  string     `json:"description,omitempty" db:"description"`
	ReleaseDate  string     `json:"release_date,omitempty" db:"release_date"`
	Rating       float32    `json:"rating,omitempty" db:"rating"`
	Genres       []string   `json:"genres,omitempty"`
	Cast         []string   `json:"cast,omitempty"`
	Directors    []string   `json:"directors,omitempty"`
	PosterURL    string     `json:"poster_url,omitempty" db:"poster_url"`
	BackdropURL  string     `json:"backdrop_url,omitempty" db:"backdrop_url"`
	TrailerURL   string     `json:"trailer_url,omitempty" db:"trailer_url"`
	LastUpdated  time.Time  `json:"last_updated" db:"last_updated"`
}

// Library represents a media library location
type Library struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Path        string      `json:"path" db:"path"`
	Type        MediaType   `json:"type" db:"type"`
	AutoScan    bool        `json:"auto_scan" db:"auto_scan"`
	ScanInterval int        `json:"scan_interval" db:"scan_interval"` // in minutes
	LastScanned time.Time   `json:"last_scanned" db:"last_scanned"`
	Created     time.Time   `json:"created" db:"created"`
	Updated     time.Time   `json:"updated" db:"updated"`
}

// StreamSession represents an active streaming session
type StreamSession struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	MediaID       uuid.UUID  `json:"media_id"`
	EpisodeID     *uuid.UUID `json:"episode_id,omitempty"`
	Profile       string     `json:"profile"`
	Position      int        `json:"position"` // current position in seconds
	Started       time.Time  `json:"started"`
	LastHeartbeat time.Time  `json:"last_heartbeat"`
	ClientInfo    ClientInfo `json:"client_info"`
}

// ClientInfo contains information about the streaming client
type ClientInfo struct {
	DeviceID   string `json:"device_id"`
	DeviceType string `json:"device_type"`
	AppVersion string `json:"app_version"`
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
}

// SearchResult represents a metadata search result
type SearchResult struct {
	ProviderID   string `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	Title        string `json:"title"`
	Year         int    `json:"year"`
	Type         string `json:"type"`
	PosterURL    string `json:"poster_url,omitempty"`
	Overview     string `json:"overview,omitempty"`
}

// EpisodeMetadata contains metadata for a single episode
type EpisodeMetadata struct {
	ID            uuid.UUID `json:"id"`
	EpisodeID     uuid.UUID `json:"episode_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	AirDate       string    `json:"air_date"`
	SeasonNumber  int       `json:"season_number"`
	EpisodeNumber int       `json:"episode_number"`
	Rating        float32   `json:"rating,omitempty"`
	GuestStars    []string  `json:"guest_stars,omitempty"`
	Directors     []string  `json:"directors,omitempty"`
	Writers       []string  `json:"writers,omitempty"`
	StillURL      string    `json:"still_url,omitempty"`
}