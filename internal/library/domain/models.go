package domain

import (
	"time"

	"github.com/google/uuid"
)

// Library represents a media library.
type Library struct {
	ID           uuid.UUID
	Name         string
	Path         string
	Type         string // movie, tv_show, music
	Enabled      bool
	ScanInterval int // seconds
	LastScanAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MetadataProviderConfig represents a metadata provider configuration.
type MetadataProviderConfig struct {
	ID           uuid.UUID
	Name         string
	ProviderType string // tmdb, tvdb, musicbrainz
	APIKey       string
	Enabled      bool
	Priority     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ScanResult represents the result of a library scan.
type ScanResult struct {
	ID           uuid.UUID
	LibraryID    uuid.UUID
	StartedAt    time.Time
	CompletedAt  *time.Time
	FilesScanned int
	FilesAdded   int
	FilesUpdated int
	FilesDeleted int
	FilesFound   int
	Status       string
	Errors       int
	ErrorMessage string
	Duration     int64 // milliseconds
}

// Media represents a media item.
type Media struct {
	ID          uuid.UUID
	LibraryID   uuid.UUID
	Title       string
	Type        string
	Path        string
	Size        int64
	Duration    int
	Resolution  string
	Codec       string
	Bitrate     int
	Added       time.Time
	Modified    time.Time
	LastScanned time.Time
}
