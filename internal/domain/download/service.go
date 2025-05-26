package download

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// Service defines the download service interface
type Service interface {
	// CreateDownload creates a new download
	CreateDownload(ctx context.Context, url string, downloadType Type, targetPath string) (*Download, error)
	
	// GetDownload retrieves a download by ID
	GetDownload(ctx context.Context, id uuid.UUID) (*Download, error)
	
	// ListDownloads lists all downloads with optional filtering
	ListDownloads(ctx context.Context, status *Status) ([]*Download, error)
	
	// StartDownload starts a download
	StartDownload(ctx context.Context, id uuid.UUID) error
	
	// PauseDownload pauses a download
	PauseDownload(ctx context.Context, id uuid.UUID) error
	
	// ResumeDownload resumes a paused download
	ResumeDownload(ctx context.Context, id uuid.UUID) error
	
	// CancelDownload cancels a download
	CancelDownload(ctx context.Context, id uuid.UUID) error
	
	// RetryDownload retries a failed download
	RetryDownload(ctx context.Context, id uuid.UUID) error
	
	// DeleteDownload deletes a download record
	DeleteDownload(ctx context.Context, id uuid.UUID) error
	
	// GetProgress gets current progress for a download
	GetProgress(ctx context.Context, id uuid.UUID) (*Progress, error)
}

// Repository defines the download repository interface
type Repository interface {
	// Save saves a download
	Save(ctx context.Context, download *Download) error
	
	// FindByID finds a download by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Download, error)
	
	// FindAll finds all downloads
	FindAll(ctx context.Context) ([]*Download, error)
	
	// FindByStatus finds downloads by status
	FindByStatus(ctx context.Context, status Status) ([]*Download, error)
	
	// Delete deletes a download
	Delete(ctx context.Context, id uuid.UUID) error
}

// Downloader defines the interface for download implementations
type Downloader interface {
	// Download starts downloading from the source
	Download(ctx context.Context, source string, destination io.Writer, progress chan<- Progress) error
	
	// Resume resumes a download from the given offset
	Resume(ctx context.Context, source string, destination io.WriteSeeker, offset int64, progress chan<- Progress) error
	
	// GetMetadata fetches metadata about the download
	GetMetadata(ctx context.Context, source string) (*Metadata, error)
}

// Validator defines the interface for file validation
type Validator interface {
	// ValidateChecksum validates a file against a checksum
	ValidateChecksum(filepath string, expectedChecksum string, checksumType string) error
	
	// CalculateChecksum calculates the checksum of a file
	CalculateChecksum(filepath string, checksumType string) (string, error)
}