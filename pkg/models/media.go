package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MediaType represents the type of media content.
type MediaType string

const (
	MediaTypeMovie  MediaType = "movie"
	MediaTypeSeries MediaType = "series"
	MediaTypeTV     MediaType = "tv" // Alias for series
	MediaTypeMusic  MediaType = "music"
)

// Media represents a media item in the library. It was previously called MediaItem in the repository.
type Media struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	LibraryID      uuid.UUID      `json:"library_id" gorm:"type:uuid;not null;index"`
	Title          string         `json:"title" gorm:"not null;index"`
	OriginalTitle  string         `json:"original_title,omitempty"`
	Type           MediaType      `json:"type" gorm:"type:varchar(50);not null;index"`
	Status         string         `json:"status,omitempty" gorm:"type:varchar(50);not null;default:'pending';index"`
	FilePath       string         `json:"file_path,omitempty" gorm:"index"`
	FileSize       int64          `json:"file_size,omitempty"`
	FileModifiedAt *time.Time     `json:"file_modified_at,omitempty"`
	Description    string         `json:"description,omitempty" gorm:"type:text"`
	ReleaseDate    *time.Time     `json:"release_date,omitempty"`
	Year           int            `json:"year,omitempty"`
	Runtime        int            `json:"runtime,omitempty"` // minutes
	Genres         []string       `json:"genres,omitempty" gorm:"type:text[]"`
	Tags           []string       `json:"tags,omitempty" gorm:"type:text[]"`
	TMDBID         int            `json:"tmdb_id,omitempty" gorm:"index"`
	IMDBID         string         `json:"imdb_id,omitempty" gorm:"type:varchar(20);index"`
	TVDBID         int            `json:"tvdb_id,omitempty" gorm:"index"`
	MusicBrainzID  *uuid.UUID     `json:"musicbrainz_id,omitempty" gorm:"type:uuid"`
	VideoCodec     string         `json:"video_codec,omitempty" gorm:"type:varchar(50)"`
	AudioCodec     string         `json:"audio_codec,omitempty" gorm:"type:varchar(50)"`
	Resolution     string         `json:"resolution,omitempty" gorm:"type:varchar(20)"`
	Bitrate        int            `json:"bitrate,omitempty"`
	PosterPath     string         `json:"poster_path,omitempty"`
	BackdropPath   string         `json:"backdrop_path,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Library  Library   `json:"-" gorm:"foreignKey:LibraryID"`
	Episodes []Episode `json:"episodes,omitempty" gorm:"foreignKey:MediaID;constraint:OnDelete:CASCADE"`
}

// Episode represents an episode of a series.
type Episode struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MediaID       uuid.UUID      `json:"media_id" gorm:"type:uuid;not null;index"`
	SeasonNumber  int            `json:"season_number" gorm:"not null;index"`
	EpisodeNumber int            `json:"episode_number" gorm:"not null;index"`
	Title         string         `json:"title"`
	Description   string         `json:"description,omitempty" gorm:"type:text"`
	AirDate       *time.Time     `json:"air_date,omitempty"`
	Runtime       int            `json:"runtime,omitempty"` // minutes
	FilePath      string         `json:"file_path,omitempty"`
	FileSize      int64          `json:"file_size,omitempty"`
	Status        string         `json:"status,omitempty" gorm:"type:varchar(50);not null;default:'missing';index"`
	VideoCodec    string         `json:"video_codec,omitempty" gorm:"type:varchar(50)"`
	AudioCodec    string         `json:"audio_codec,omitempty" gorm:"type:varchar(50)"`
	Resolution    string         `json:"resolution,omitempty" gorm:"type:varchar(20)"`
	Bitrate       int            `json:"bitrate,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Media Media `json:"-" gorm:"foreignKey:MediaID"`
}

// Library represents a media library location.
type Library struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string         `json:"name" gorm:"uniqueIndex;not null"`
	Path         string         `json:"path" gorm:"uniqueIndex;not null"`
	Type         MediaType      `json:"type" gorm:"type:varchar(50);not null"`
	Enabled      bool           `json:"enabled" gorm:"default:true"`
	ScanInterval int            `json:"scan_interval" gorm:"default:3600"` // seconds
	LastScanAt   *time.Time     `json:"last_scan_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	MediaItems  []Media       `json:"-" gorm:"foreignKey:LibraryID;constraint:OnDelete:CASCADE"`
	ScanHistory []ScanHistory `json:"-" gorm:"foreignKey:LibraryID;constraint:OnDelete:CASCADE"`
}

// MetadataProvider represents a metadata provider configuration.
type MetadataProvider struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string         `json:"name" gorm:"uniqueIndex;not null"`
	ProviderType string         `json:"provider_type" gorm:"type:varchar(50);not null"` // tmdb, tvdb, musicbrainz
	APIKey       string         `json:"-" gorm:"type:text"`                             // Should be encrypted
	Enabled      bool           `json:"enabled" gorm:"default:true"`
	Priority     int            `json:"priority" gorm:"default:0"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// ScanHistory represents a library scan event.
type ScanHistory struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	LibraryID    uuid.UUID      `json:"library_id" gorm:"type:uuid;not null;index"`
	StartedAt    time.Time      `json:"started_at" gorm:"not null;default:CURRENT_TIMESTAMP;index"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	FilesScanned int            `json:"files_scanned" gorm:"default:0"`
	FilesAdded   int            `json:"files_added" gorm:"default:0"`
	FilesUpdated int            `json:"files_updated" gorm:"default:0"`
	FilesDeleted int            `json:"files_deleted" gorm:"default:0"`
	ErrorMessage string         `json:"error_message,omitempty" gorm:"type:text"`

	// Relationships
	Library Library `json:"-" gorm:"foreignKey:LibraryID"`
}

// TableName customizations.
func (Library) TableName() string {
	return "libraries"
}

func (Media) TableName() string {
	return "media_items"
}

func (Episode) TableName() string {
	return "episodes"
}

func (MetadataProvider) TableName() string {
	return "metadata_providers"
}

func (ScanHistory) TableName() string {
	return "scan_history"
}

// The structs below were in the original file but are not directly related to library GORM models.
// They are likely used for other purposes like API responses, so I'm keeping them.
// I've removed the `db` tags as they are not needed.

// Metadata contains enriched metadata for media items.
type Metadata struct {
	ID          uuid.UUID `json:"id"`
	MediaID     uuid.UUID `json:"media_id"`
	Title       string    `json:"title,omitempty"`
	IMDBID      string    `json:"imdb_id,omitempty"`
	TMDBID      string    `json:"tmdb_id,omitempty"`
	TVDBID      string    `json:"tvdb_id,omitempty"`
	Description string    `json:"description,omitempty"`
	ReleaseDate string    `json:"release_date,omitempty"`
	Rating      float32   `json:"rating,omitempty"`
	Genres      []string  `json:"genres,omitempty"`
	Cast        []string  `json:"cast,omitempty"`
	Directors   []string  `json:"directors,omitempty"`
	PosterURL   string    `json:"poster_url,omitempty"`
	BackdropURL string    `json:"backdrop_url,omitempty"`
	TrailerURL  string    `json:"trailer_url,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// StreamSession represents an active streaming session.
type StreamSession struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	MediaID       uuid.UUID  `json:"media_id"`
	EpisodeID     *uuid.UUID `json:"episode_id,omitempty"`
	Profile       string     `json:"profile"`
	Position      int        `json:"position"` // in seconds
	Started       time.Time  `json:"started"`
	LastHeartbeat time.Time  `json:"last_heartbeat"`
	ClientInfo    ClientInfo `json:"client_info"`
}

// ClientInfo contains information about the streaming client.
type ClientInfo struct {
	DeviceID   string `json:"device_id"`
	DeviceType string `json:"device_type"`
	AppVersion string `json:"app_version"`
	IP         string `json:"ip"`
	UserAgent  string `json:"user_agent"`
}

// SearchResult represents a metadata search result.
type SearchResult struct {
	ProviderID   string `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	Title        string `json:"title"`
	Year         int    `json:"year"`
	Type         string `json:"type"`
	PosterURL    string `json:"poster_url,omitempty"`
	Overview     string `json:"overview,omitempty"`
}

// EpisodeMetadata contains metadata for a single episode.
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
