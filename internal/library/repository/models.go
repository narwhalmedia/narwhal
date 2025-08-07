package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Library represents a media library in the database.
type Library struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string    `gorm:"uniqueIndex;not null"`
	Path         string    `gorm:"uniqueIndex;not null"`
	MediaType    string    `gorm:"type:varchar(50);not null"`
	Enabled      bool      `gorm:"default:true"`
	ScanInterval int       `gorm:"default:3600"` // seconds
	LastScanAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relationships
	MediaItems  []MediaItem   `gorm:"foreignKey:LibraryID;constraint:OnDelete:CASCADE"`
	ScanHistory []ScanHistory `gorm:"foreignKey:LibraryID;constraint:OnDelete:CASCADE"`
}

// MediaItem represents a media file in the database.
type MediaItem struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	LibraryID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Title          string    `gorm:"not null;index"`
	OriginalTitle  string
	MediaType      string `gorm:"type:varchar(50);not null;index"`
	Status         string `gorm:"type:varchar(50);not null;default:'pending';index"`
	FilePath       string `gorm:"index"`
	FileSize       int64
	FileModifiedAt *time.Time

	// Metadata
	Description string `gorm:"type:text"`
	ReleaseDate *time.Time
	Runtime     int      // minutes
	Genres      []string `gorm:"type:text[]"`
	Tags        []string `gorm:"type:text[]"`

	// External IDs
	TMDBID        int        `gorm:"index"`
	IMDBID        string     `gorm:"type:varchar(20);index"`
	TVDBID        int        `gorm:"index"`
	MusicBrainzID *uuid.UUID `gorm:"type:uuid"`

	// Media info
	VideoCodec string `gorm:"type:varchar(50)"`
	AudioCodec string `gorm:"type:varchar(50)"`
	Resolution string `gorm:"type:varchar(20)"`
	Bitrate    int

	// Artwork
	PosterPath   string
	BackdropPath string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relationships
	Library  Library   `gorm:"foreignKey:LibraryID"`
	Episodes []Episode `gorm:"foreignKey:MediaID;constraint:OnDelete:CASCADE"`
}

// Episode represents a TV show episode.
type Episode struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MediaID       uuid.UUID `gorm:"type:uuid;not null;index"`
	SeasonNumber  int       `gorm:"not null;index"`
	EpisodeNumber int       `gorm:"not null;index"`
	Title         string
	Description   string `gorm:"type:text"`
	AirDate       *time.Time
	Runtime       int // minutes
	FilePath      string
	FileSize      int64
	Status        string `gorm:"type:varchar(50);not null;default:'missing';index"`

	// Media info
	VideoCodec string `gorm:"type:varchar(50)"`
	AudioCodec string `gorm:"type:varchar(50)"`
	Resolution string `gorm:"type:varchar(20)"`
	Bitrate    int

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relationships
	Media MediaItem `gorm:"foreignKey:MediaID"`
}

// MetadataProvider represents a metadata provider configuration.
type MetadataProvider struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string    `gorm:"uniqueIndex;not null"`
	ProviderType string    `gorm:"type:varchar(50);not null"` // tmdb, tvdb, musicbrainz
	APIKey       string    `gorm:"type:text"`                 // Should be encrypted
	Enabled      bool      `gorm:"default:true"`
	Priority     int       `gorm:"default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// ScanHistory represents a library scan event.
type ScanHistory struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	LibraryID    uuid.UUID `gorm:"type:uuid;not null;index"`
	StartedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index"`
	CompletedAt  *time.Time
	FilesScanned int    `gorm:"default:0"`
	FilesAdded   int    `gorm:"default:0"`
	FilesUpdated int    `gorm:"default:0"`
	FilesDeleted int    `gorm:"default:0"`
	ErrorMessage string `gorm:"type:text"`

	// Relationships
	Library Library `gorm:"foreignKey:LibraryID"`
}

// BeforeCreate hook for Episode to ensure unique constraint.
func (e *Episode) BeforeCreate(tx *gorm.DB) error {
	// GORM doesn't support composite unique indexes well, so we'll handle it in the migration
	return nil
}

// TableName customizations if needed.
func (Library) TableName() string {
	return "libraries"
}

func (MediaItem) TableName() string {
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
