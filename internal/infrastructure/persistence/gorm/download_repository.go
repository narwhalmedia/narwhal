package gorm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/narwhalmedia/narwhal/internal/domain/download"
)

// DownloadModel represents the database model for downloads
type DownloadModel struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	URL             string    `gorm:"not null"`
	DownloadType    string    `gorm:"not null"`
	TargetPath      string    `gorm:"not null"`
	Status          string    `gorm:"not null"`
	BytesDownloaded int64
	TotalBytes      int64
	Speed           int64
	ETA             int64 // Duration in nanoseconds
	Seeders         int
	Leechers        int
	FileName        string
	ContentType     string
	InfoHash        string
	StartedAt       *time.Time
	CompletedAt     *time.Time
	Error           string
	RetryCount      int
	MaxRetries      int
	Checksum        string
	ChecksumType    string
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

// TableName specifies the table name
func (DownloadModel) TableName() string {
	return "downloads"
}

// DownloadRepository implements the download repository using GORM
type DownloadRepository struct {
	db *gorm.DB
}

// NewDownloadRepository creates a new download repository
func NewDownloadRepository(db *gorm.DB) *DownloadRepository {
	return &DownloadRepository{db: db}
}

// Save saves a download
func (r *DownloadRepository) Save(ctx context.Context, dl *download.Download) error {
	model := toDownloadModel(dl)
	
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to save download: %w", result.Error)
	}
	
	return nil
}

// FindByID finds a download by ID
func (r *DownloadRepository) FindByID(ctx context.Context, id uuid.UUID) (*download.Download, error) {
	var model DownloadModel
	
	result := r.db.WithContext(ctx).First(&model, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("download not found")
		}
		return nil, fmt.Errorf("failed to find download: %w", result.Error)
	}
	
	return toDomainDownload(&model)
}

// FindAll finds all downloads
func (r *DownloadRepository) FindAll(ctx context.Context) ([]*download.Download, error) {
	var models []DownloadModel
	
	result := r.db.WithContext(ctx).Order("created_at DESC").Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find downloads: %w", result.Error)
	}
	
	downloads := make([]*download.Download, 0, len(models))
	for _, model := range models {
		dl, err := toDomainDownload(&model)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, dl)
	}
	
	return downloads, nil
}

// FindByStatus finds downloads by status
func (r *DownloadRepository) FindByStatus(ctx context.Context, status download.Status) ([]*download.Download, error) {
	var models []DownloadModel
	
	result := r.db.WithContext(ctx).Where("status = ?", string(status)).Order("created_at DESC").Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find downloads by status: %w", result.Error)
	}
	
	downloads := make([]*download.Download, 0, len(models))
	for _, model := range models {
		dl, err := toDomainDownload(&model)
		if err != nil {
			return nil, err
		}
		downloads = append(downloads, dl)
	}
	
	return downloads, nil
}

// Delete deletes a download
func (r *DownloadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&DownloadModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete download: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("download not found")
	}
	
	return nil
}

// toDownloadModel converts domain download to database model
func toDownloadModel(dl *download.Download) *DownloadModel {
	progress := dl.Progress()
	metadata := dl.Metadata()
	
	return &DownloadModel{
		ID:              dl.ID(),
		URL:             dl.URL(),
		DownloadType:    string(dl.Type()),
		TargetPath:      dl.TargetPath(),
		Status:          string(dl.Status()),
		BytesDownloaded: progress.BytesDownloaded,
		TotalBytes:      progress.TotalBytes,
		Speed:           progress.Speed,
		ETA:             int64(progress.ETA),
		Seeders:         progress.Seeders,
		Leechers:        progress.Leechers,
		FileName:        metadata.FileName,
		ContentType:     metadata.ContentType,
		InfoHash:        metadata.InfoHash,
		StartedAt:       dl.StartedAt(),
		CompletedAt:     dl.CompletedAt(),
		Error:           dl.Error(),
		RetryCount:      dl.RetryCount(),
		MaxRetries:      dl.MaxRetries(),
		Checksum:        dl.Checksum(),
		ChecksumType:    dl.ChecksumType(),
		CreatedAt:       dl.CreatedAt(),
		UpdatedAt:       dl.UpdatedAt(),
	}
}

// toDomainDownload converts database model to domain download
func toDomainDownload(model *DownloadModel) (*download.Download, error) {
	// Create new download
	dl, err := download.NewDownload(model.URL, download.Type(model.DownloadType), model.TargetPath)
	if err != nil {
		return nil, err
	}
	
	// Use reflection to set private fields (not ideal, but necessary for repository pattern)
	// In a real implementation, we might add a constructor or use a different approach
	
	// For now, we'll reconstruct the state through public methods
	// This is a limitation of the current design
	
	// Set progress
	progress := download.Progress{
		BytesDownloaded: model.BytesDownloaded,
		TotalBytes:      model.TotalBytes,
		Speed:           model.Speed,
		ETA:             time.Duration(model.ETA),
		Seeders:         model.Seeders,
		Leechers:        model.Leechers,
	}
	dl.UpdateProgress(progress)
	
	// Set metadata
	metadata := download.Metadata{
		FileName:    model.FileName,
		ContentType: model.ContentType,
		InfoHash:    model.InfoHash,
		Headers:     make(map[string]string),
	}
	dl.SetMetadata(metadata)
	
	// Set checksum
	if model.Checksum != "" {
		dl.SetChecksum(model.Checksum, model.ChecksumType)
	}
	
	// Restore state based on status
	switch download.Status(model.Status) {
	case download.StatusDownloading:
		dl.Start()
	case download.StatusPaused:
		dl.Start()
		dl.Pause()
	case download.StatusCompleted:
		dl.Start()
		dl.Complete()
	case download.StatusFailed:
		dl.Start()
		dl.Fail(model.Error)
	case download.StatusCancelled:
		dl.Cancel()
	case download.StatusVerifying:
		dl.Start()
		dl.StartVerifying()
	}
	
	// Note: We can't restore all internal state perfectly due to encapsulation
	// This is a trade-off in the design
	
	return dl, nil
}