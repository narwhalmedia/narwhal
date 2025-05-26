package gorm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/narwhalmedia/narwhal/internal/domain/transcode"
)

type TranscodeJob struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key"`
	InputPath      string    `gorm:"not null"`
	OutputPath     string    `gorm:"not null"`
	Profile        string    `gorm:"not null"`
	Status         string    `gorm:"not null"`
	Progress       int
	StartedAt      *time.Time
	CompletedAt    *time.Time
	ErrorMessage   string
	Options        string `gorm:"type:json"`
	Metadata       string `gorm:"type:json"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (TranscodeJob) TableName() string {
	return "transcode_jobs"
}

type TranscodeJobRepository struct {
	db *gorm.DB
}

func NewTranscodeJobRepository(db *gorm.DB) (*TranscodeJobRepository, error) {
	if err := db.AutoMigrate(&TranscodeJob{}); err != nil {
		return nil, fmt.Errorf("failed to migrate transcode_job table: %w", err)
	}

	return &TranscodeJobRepository{db: db}, nil
}

func (r *TranscodeJobRepository) Save(ctx context.Context, job *transcode.Job) error {
	model, err := r.toModel(job)
	if err != nil {
		return fmt.Errorf("failed to convert to model: %w", err)
	}

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to save job: %w", result.Error)
	}

	return nil
}

func (r *TranscodeJobRepository) FindByID(ctx context.Context, id uuid.UUID) (*transcode.Job, error) {
	var model TranscodeJob
	
	result := r.db.WithContext(ctx).First(&model, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, transcode.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to find job: %w", result.Error)
	}

	return r.toDomain(&model)
}

func (r *TranscodeJobRepository) List(ctx context.Context, status transcode.Status, limit int) ([]*transcode.Job, error) {
	var models []TranscodeJob
	
	query := r.db.WithContext(ctx)
	if status != "" {
		query = query.Where("status = ?", string(status))
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	query = query.Order("created_at DESC")

	result := query.Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", result.Error)
	}

	jobs := make([]*transcode.Job, len(models))
	for i, model := range models {
		job, err := r.toDomain(&model)
		if err != nil {
			return nil, fmt.Errorf("failed to convert job %s: %w", model.ID, err)
		}
		jobs[i] = job
	}

	return jobs, nil
}

func (r *TranscodeJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&TranscodeJob{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete job: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return transcode.ErrJobNotFound
	}

	return nil
}

func (r *TranscodeJobRepository) toModel(job *transcode.Job) (*TranscodeJob, error) {
	optionsJSON, err := json.Marshal(job.Options())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	metadataJSON, err := json.Marshal(job.Metadata())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	model := &TranscodeJob{
		ID:           job.ID(),
		InputPath:    job.InputPath(),
		OutputPath:   job.OutputPath(),
		Profile:      string(job.Profile()),
		Status:       string(job.Status()),
		Options:      string(optionsJSON),
		Metadata:     string(metadataJSON),
		CreatedAt:    job.CreatedAt(),
		UpdatedAt:    job.UpdatedAt(),
	}

	// Handle progress
	progress := job.Progress()
	model.Progress = int(progress.Percent)

	// Handle timestamps
	if job.StartedAt() != nil && !job.StartedAt().IsZero() {
		model.StartedAt = job.StartedAt()
	}
	if job.CompletedAt() != nil && !job.CompletedAt().IsZero() {
		model.CompletedAt = job.CompletedAt()
	}

	// Handle error
	if job.Error() != "" {
		model.ErrorMessage = job.Error()
	}

	return model, nil
}

func (r *TranscodeJobRepository) toDomain(model *TranscodeJob) (*transcode.Job, error) {
	var options transcode.Options
	if model.Options != "" {
		if err := json.Unmarshal([]byte(model.Options), &options); err != nil {
			return nil, fmt.Errorf("failed to unmarshal options: %w", err)
		}
	}

	var metadata transcode.Metadata
	if model.Metadata != "" {
		if err := json.Unmarshal([]byte(model.Metadata), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	job := transcode.NewJobFromRepository(
		model.ID,
		model.InputPath,
		model.OutputPath,
		transcode.Profile(model.Profile),
		transcode.Status(model.Status),
		options,
		metadata,
		model.CreatedAt,
		model.UpdatedAt,
	)

	// Set progress if available
	if model.Progress > 0 {
		progress := transcode.Progress{
			Percent: float64(model.Progress),
		}
		job.SetProgress(progress)
	}

	// Set timestamps
	if model.StartedAt != nil {
		job.SetStartedAt(*model.StartedAt)
	}
	if model.CompletedAt != nil {
		job.SetCompletedAt(*model.CompletedAt)
	}

	// Set error
	if model.ErrorMessage != "" {
		job.SetError(model.ErrorMessage)
	}

	return job, nil
}