package models

import (
	"time"

	"github.com/google/uuid"
)

// DownloadStatus represents the status of a download
type DownloadStatus string

const (
	DownloadStatusPending     DownloadStatus = "pending"
	DownloadStatusQueued      DownloadStatus = "queued"
	DownloadStatusDownloading DownloadStatus = "downloading"
	DownloadStatusCompleted   DownloadStatus = "completed"
	DownloadStatusFailed      DownloadStatus = "failed"
	DownloadStatusCancelled   DownloadStatus = "cancelled"
)

// Download represents a download task
type Download struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	Title           string         `json:"title" db:"title"`
	Type            MediaType      `json:"type" db:"type"`
	IndexerID       string         `json:"indexer_id" db:"indexer_id"`
	DownloadURL     string         `json:"download_url" db:"download_url"`
	Size            int64          `json:"size" db:"size"`
	Status          DownloadStatus `json:"status" db:"status"`
	Progress        float32        `json:"progress" db:"progress"`
	DownloadSpeed   int64          `json:"download_speed" db:"download_speed"`
	ETA             int            `json:"eta" db:"eta"` // in seconds
	DownloadClient  string         `json:"download_client" db:"download_client"`
	OutputPath      string         `json:"output_path" db:"output_path"`
	Priority        int            `json:"priority" db:"priority"`
	RetryCount      int            `json:"retry_count" db:"retry_count"`
	Error           string         `json:"error,omitempty" db:"error"`
	Started         *time.Time     `json:"started,omitempty" db:"started"`
	Completed       *time.Time     `json:"completed,omitempty" db:"completed"`
	Created         time.Time      `json:"created" db:"created"`
	Updated         time.Time      `json:"updated" db:"updated"`
}

// Release represents a release from an indexer
type Release struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	IndexerID    string    `json:"indexer_id"`
	IndexerName  string    `json:"indexer_name"`
	Size         int64     `json:"size"`
	PublishDate  time.Time `json:"publish_date"`
	DownloadURL  string    `json:"download_url"`
	InfoURL      string    `json:"info_url,omitempty"`
	Seeders      int       `json:"seeders,omitempty"`
	Leechers     int       `json:"leechers,omitempty"`
	Quality      Quality   `json:"quality"`
	SceneSource  bool      `json:"scene_source"`
	FreeLeech    bool      `json:"free_leech"`
}

// Quality represents the quality profile of a release
type Quality struct {
	Resolution string `json:"resolution"` // 1080p, 720p, etc.
	Source     string `json:"source"`     // BluRay, WEB-DL, etc.
	Codec      string `json:"codec"`      // x264, x265, etc.
	BitDepth   int    `json:"bit_depth"`  // 8, 10
	HDR        bool   `json:"hdr"`
	Score      int    `json:"score"` // Quality score for comparison
}

// QualityProfile represents a quality profile for automatic selection
type QualityProfile struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	MinScore           int       `json:"min_score" db:"min_score"`
	MaxScore           int       `json:"max_score" db:"max_score"`
	PreferredScore     int       `json:"preferred_score" db:"preferred_score"`
	UpgradeAllowed     bool      `json:"upgrade_allowed" db:"upgrade_allowed"`
	PreferredKeywords  []string  `json:"preferred_keywords"`
	RequiredKeywords   []string  `json:"required_keywords"`
	IgnoredKeywords    []string  `json:"ignored_keywords"`
}

// DownloadHistory represents the history of download attempts
type DownloadHistory struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	DownloadID   uuid.UUID      `json:"download_id" db:"download_id"`
	Status       DownloadStatus `json:"status" db:"status"`
	Message      string         `json:"message" db:"message"`
	Timestamp    time.Time      `json:"timestamp" db:"timestamp"`
}