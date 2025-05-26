package download

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Status represents the status of a download
type Status string

const (
	StatusPending     Status = "pending"
	StatusDownloading Status = "downloading"
	StatusPaused      Status = "paused"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
	StatusVerifying   Status = "verifying"
)

// Type represents the type of download
type Type string

const (
	TypeHTTP    Type = "http"
	TypeTorrent Type = "torrent"
	TypeUsenet  Type = "usenet"
)

// Download represents a download task
type Download struct {
	id              uuid.UUID
	url             string
	downloadType    Type
	targetPath      string
	status          Status
	progress        Progress
	metadata        Metadata
	startedAt       *time.Time
	completedAt     *time.Time
	error           string
	retryCount      int
	maxRetries      int
	checksum        string
	checksumType    string
	createdAt       time.Time
	updatedAt       time.Time
}

// Progress tracks download progress
type Progress struct {
	BytesDownloaded int64
	TotalBytes      int64
	Speed           int64 // bytes per second
	ETA             time.Duration
	Seeders         int // for torrents
	Leechers        int // for torrents
}

// Metadata contains additional download information
type Metadata struct {
	FileName    string
	ContentType string
	Headers     map[string]string
	InfoHash    string // for torrents
	NZBInfo     *NZBInfo
}

// NZBInfo contains Usenet-specific information
type NZBInfo struct {
	Subject     string
	Groups      []string
	Segments    int
	TotalSize   int64
	PostDate    time.Time
}

// NewDownload creates a new download
func NewDownload(url string, downloadType Type, targetPath string) (*Download, error) {
	if url == "" {
		return nil, fmt.Errorf("download URL is required")
	}
	if targetPath == "" {
		return nil, fmt.Errorf("target path is required")
	}

	now := time.Now()
	return &Download{
		id:           uuid.New(),
		url:          url,
		downloadType: downloadType,
		targetPath:   targetPath,
		status:       StatusPending,
		progress:     Progress{},
		metadata:     Metadata{Headers: make(map[string]string)},
		maxRetries:   3,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// Getters
func (d *Download) ID() uuid.UUID          { return d.id }
func (d *Download) URL() string            { return d.url }
func (d *Download) Type() Type             { return d.downloadType }
func (d *Download) TargetPath() string     { return d.targetPath }
func (d *Download) Status() Status         { return d.status }
func (d *Download) Progress() Progress     { return d.progress }
func (d *Download) Metadata() Metadata     { return d.metadata }
func (d *Download) StartedAt() *time.Time  { return d.startedAt }
func (d *Download) CompletedAt() *time.Time { return d.completedAt }
func (d *Download) Error() string          { return d.error }
func (d *Download) RetryCount() int        { return d.retryCount }
func (d *Download) MaxRetries() int        { return d.maxRetries }
func (d *Download) Checksum() string       { return d.checksum }
func (d *Download) ChecksumType() string   { return d.checksumType }
func (d *Download) CreatedAt() time.Time   { return d.createdAt }
func (d *Download) UpdatedAt() time.Time   { return d.updatedAt }

// Start marks the download as started
func (d *Download) Start() error {
	if d.status != StatusPending && d.status != StatusPaused {
		return fmt.Errorf("cannot start download in status %s", d.status)
	}

	now := time.Now()
	d.status = StatusDownloading
	d.startedAt = &now
	d.updatedAt = now
	return nil
}

// Pause pauses the download
func (d *Download) Pause() error {
	if d.status != StatusDownloading {
		return fmt.Errorf("cannot pause download in status %s", d.status)
	}

	d.status = StatusPaused
	d.updatedAt = time.Now()
	return nil
}

// Resume resumes a paused download
func (d *Download) Resume() error {
	if d.status != StatusPaused {
		return fmt.Errorf("cannot resume download in status %s", d.status)
	}

	d.status = StatusDownloading
	d.updatedAt = time.Now()
	return nil
}

// Cancel cancels the download
func (d *Download) Cancel() {
	d.status = StatusCancelled
	d.updatedAt = time.Now()
}

// Complete marks the download as completed
func (d *Download) Complete() error {
	if d.status != StatusDownloading && d.status != StatusVerifying {
		return fmt.Errorf("cannot complete download in status %s", d.status)
	}

	now := time.Now()
	d.status = StatusCompleted
	d.completedAt = &now
	d.updatedAt = now
	return nil
}

// Fail marks the download as failed
func (d *Download) Fail(err string) {
	d.status = StatusFailed
	d.error = err
	d.updatedAt = time.Now()
}

// UpdateProgress updates download progress
func (d *Download) UpdateProgress(progress Progress) {
	d.progress = progress
	d.updatedAt = time.Now()
}

// SetMetadata sets download metadata
func (d *Download) SetMetadata(metadata Metadata) {
	d.metadata = metadata
	d.updatedAt = time.Now()
}

// SetChecksum sets the expected checksum
func (d *Download) SetChecksum(checksum, checksumType string) {
	d.checksum = checksum
	d.checksumType = checksumType
	d.updatedAt = time.Now()
}

// IncrementRetry increments the retry count
func (d *Download) IncrementRetry() bool {
	d.retryCount++
	d.updatedAt = time.Now()
	return d.retryCount <= d.maxRetries
}

// StartVerifying marks the download as verifying
func (d *Download) StartVerifying() error {
	if d.status != StatusDownloading {
		return fmt.Errorf("cannot start verifying in status %s", d.status)
	}

	d.status = StatusVerifying
	d.updatedAt = time.Now()
	return nil
}

// CalculateETA calculates estimated time of arrival
func (p *Progress) CalculateETA() time.Duration {
	if p.Speed <= 0 || p.BytesDownloaded >= p.TotalBytes {
		return 0
	}

	remainingBytes := p.TotalBytes - p.BytesDownloaded
	seconds := remainingBytes / p.Speed
	return time.Duration(seconds) * time.Second
}

// PercentComplete returns the completion percentage
func (p *Progress) PercentComplete() float64 {
	if p.TotalBytes <= 0 {
		return 0
	}
	return float64(p.BytesDownloaded) / float64(p.TotalBytes) * 100
}