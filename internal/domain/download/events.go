package download

import (
	"time"

	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
)

// DownloadCreated is emitted when a download is created
type DownloadCreated struct {
	domainevents.BaseEvent
	URL          string `json:"url"`
	DownloadType string `json:"download_type"`
	TargetPath   string `json:"target_path"`
}

// NewDownloadCreated creates a new DownloadCreated event
func NewDownloadCreated(download *Download) *DownloadCreated {
	return &DownloadCreated{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadCreated",
			1,
		),
		URL:          download.URL(),
		DownloadType: string(download.Type()),
		TargetPath:   download.TargetPath(),
	}
}

// DownloadStarted is emitted when a download starts
type DownloadStarted struct {
	domainevents.BaseEvent
	URL string `json:"url"`
}

// NewDownloadStarted creates a new DownloadStarted event
func NewDownloadStarted(download *Download) *DownloadStarted {
	return &DownloadStarted{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadStarted",
			1,
		),
		URL: download.URL(),
	}
}

// DownloadProgress is emitted periodically with download progress
type DownloadProgress struct {
	domainevents.BaseEvent
	BytesDownloaded int64   `json:"bytes_downloaded"`
	TotalBytes      int64   `json:"total_bytes"`
	Speed           int64   `json:"speed"`
	PercentComplete float64 `json:"percent_complete"`
}

// NewDownloadProgress creates a new DownloadProgress event
func NewDownloadProgress(download *Download) *DownloadProgress {
	progress := download.Progress()
	return &DownloadProgress{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadProgress",
			1,
		),
		BytesDownloaded: progress.BytesDownloaded,
		TotalBytes:      progress.TotalBytes,
		Speed:           progress.Speed,
		PercentComplete: progress.PercentComplete(),
	}
}

// DownloadCompleted is emitted when a download completes
type DownloadCompleted struct {
	domainevents.BaseEvent
	FilePath      string        `json:"file_path"`
	FileSize      int64         `json:"file_size"`
	Duration      time.Duration `json:"duration"`
	Checksum      string        `json:"checksum,omitempty"`
	ChecksumType  string        `json:"checksum_type,omitempty"`
}

// NewDownloadCompleted creates a new DownloadCompleted event
func NewDownloadCompleted(download *Download) *DownloadCompleted {
	var duration time.Duration
	if download.StartedAt() != nil && download.CompletedAt() != nil {
		duration = download.CompletedAt().Sub(*download.StartedAt())
	}

	return &DownloadCompleted{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadCompleted",
			1,
		),
		FilePath:     download.TargetPath(),
		FileSize:     download.Progress().TotalBytes,
		Duration:     duration,
		Checksum:     download.Checksum(),
		ChecksumType: download.ChecksumType(),
	}
}

// DownloadFailed is emitted when a download fails
type DownloadFailed struct {
	domainevents.BaseEvent
	Error      string `json:"error"`
	RetryCount int    `json:"retry_count"`
	CanRetry   bool   `json:"can_retry"`
}

// NewDownloadFailed creates a new DownloadFailed event
func NewDownloadFailed(download *Download, canRetry bool) *DownloadFailed {
	return &DownloadFailed{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadFailed",
			1,
		),
		Error:      download.Error(),
		RetryCount: download.RetryCount(),
		CanRetry:   canRetry,
	}
}

// DownloadPaused is emitted when a download is paused
type DownloadPaused struct {
	domainevents.BaseEvent
	BytesDownloaded int64 `json:"bytes_downloaded"`
}

// NewDownloadPaused creates a new DownloadPaused event
func NewDownloadPaused(download *Download) *DownloadPaused {
	return &DownloadPaused{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadPaused",
			1,
		),
		BytesDownloaded: download.Progress().BytesDownloaded,
	}
}

// DownloadResumed is emitted when a download is resumed
type DownloadResumed struct {
	domainevents.BaseEvent
	BytesDownloaded int64 `json:"bytes_downloaded"`
}

// NewDownloadResumed creates a new DownloadResumed event
func NewDownloadResumed(download *Download) *DownloadResumed {
	return &DownloadResumed{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadResumed",
			1,
		),
		BytesDownloaded: download.Progress().BytesDownloaded,
	}
}

// DownloadCancelled is emitted when a download is cancelled
type DownloadCancelled struct {
	domainevents.BaseEvent
}

// NewDownloadCancelled creates a new DownloadCancelled event
func NewDownloadCancelled(download *Download) *DownloadCancelled {
	return &DownloadCancelled{
		BaseEvent: domainevents.NewBaseEvent(
			download.ID(),
			"Download",
			"DownloadCancelled",
			1,
		),
	}
}