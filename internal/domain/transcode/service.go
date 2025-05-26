package transcode

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// Service defines the transcode service interface
type Service interface {
	// CreateJob creates a new transcode job
	CreateJob(ctx context.Context, inputPath, outputPath string, profile Profile) (*Job, error)
	
	// CreateJobWithOptions creates a new transcode job with custom options
	CreateJobWithOptions(ctx context.Context, inputPath, outputPath string, profile Profile, opts JobOptions) (*Job, error)
	
	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, id uuid.UUID) (*Job, error)
	
	// ListJobs lists all jobs with optional filtering
	ListJobs(ctx context.Context, status *Status) ([]*Job, error)
	
	// StartJob starts a transcode job
	StartJob(ctx context.Context, id uuid.UUID) error
	
	// CancelJob cancels a job
	CancelJob(ctx context.Context, id uuid.UUID) error
	
	// RetryJob retries a failed job
	RetryJob(ctx context.Context, id uuid.UUID) error
	
	// DeleteJob deletes a job record
	DeleteJob(ctx context.Context, id uuid.UUID) error
	
	// GetProgress gets current progress for a job
	GetProgress(ctx context.Context, id uuid.UUID) (*Progress, error)
	
	// GetVariants gets HLS variants for a completed job
	GetVariants(ctx context.Context, id uuid.UUID) ([]Variant, error)
}

// Repository defines the transcode repository interface
type Repository interface {
	// Save saves a job
	Save(ctx context.Context, job *Job) error
	
	// FindByID finds a job by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Job, error)
	
	// List lists jobs with optional status filter
	List(ctx context.Context, status Status, limit int) ([]*Job, error)
	
	// Delete deletes a job
	Delete(ctx context.Context, id uuid.UUID) error
}

// Transcoder defines the interface for transcode implementations
type Transcoder interface {
	// Transcode performs the actual transcoding
	Transcode(ctx context.Context, job *Job, progress chan<- Progress) error
	
	// Cancel cancels an ongoing transcode
	Cancel(ctx context.Context, jobID uuid.UUID) error
	
	// GetCapabilities returns transcoder capabilities
	GetCapabilities() Capabilities
}

// VariantConfig defines configuration for an HLS variant
type VariantConfig struct {
	Name       string
	Resolution string
	VideoBitrate string
	AudioBitrate string
	MaxBitrate string
	BufferSize string
}

// StorageBackend defines the interface for output storage
type StorageBackend interface {
	// Store stores data from a reader
	Store(ctx context.Context, key string, reader io.Reader) error
	
	// Retrieve retrieves data to a reader
	Retrieve(ctx context.Context, key string) (io.ReadCloser, error)
	
	// Delete deletes stored data
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)
	
	// GetURL gets a public URL for a file
	GetURL(ctx context.Context, key string) (string, error)
}

// ProgressParser parses FFmpeg progress output
type ProgressParser interface {
	// Parse parses a line of FFmpeg output
	Parse(line string) (*Progress, error)
}

// Capabilities describes what a transcoder can do
type Capabilities struct {
	SupportedProfiles    []Profile
	SupportedCodecs      []string
	MaxResolution        Resolution
	HardwareAcceleration bool
}

// Resolution represents video resolution
type Resolution struct {
	Width  int
	Height int
}

// HLSVariant represents an HLS variant stream
type HLSVariant struct {
	Name    string
	Width   int
	Height  int
	Bitrate int // in kbps
	CRF     int
}

// DefaultHLSVariants returns default HLS variant configurations
var DefaultHLSVariants = []HLSVariant{
	{Name: "1080p", Width: 1920, Height: 1080, Bitrate: 5000, CRF: 22},
	{Name: "720p", Width: 1280, Height: 720, Bitrate: 2800, CRF: 23},
	{Name: "480p", Width: 854, Height: 480, Bitrate: 1400, CRF: 23},
	{Name: "360p", Width: 640, Height: 360, Bitrate: 800, CRF: 23},
}