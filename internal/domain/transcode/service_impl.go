package transcode

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
	"github.com/narwhalmedia/narwhal/internal/events"
)

type serviceImpl struct {
	repo       Repository
	transcoder Transcoder
	storage    StorageBackend
	eventBus   events.EventBus
	logger     *zap.Logger
}

// NewService creates a new transcode service
func NewService(
	repo Repository,
	transcoder Transcoder,
	storage StorageBackend,
	eventBus events.EventBus,
	logger *zap.Logger,
) Service {
	return &serviceImpl{
		repo:       repo,
		transcoder: transcoder,
		storage:    storage,
		eventBus:   eventBus,
		logger:     logger,
	}
}

func (s *serviceImpl) CreateJob(ctx context.Context, inputPath, outputPath string, profile Profile) (*Job, error) {
	// Validate profile is supported
	capabilities := s.transcoder.GetCapabilities()
	supported := false
	for _, p := range capabilities.SupportedProfiles {
		if p == profile {
			supported = true
			break
		}
	}
	if !supported {
		return nil, ErrUnsupportedProfile
	}

	// Create new job
	job, err := NewJob(inputPath, outputPath, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Save to repository
	if err := s.repo.Save(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Publish event
	event := &JobCreated{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"transcode",
			"transcode.job.created",
			1,
		),
		InputPath:  inputPath,
		OutputPath: outputPath,
		Profile:    string(profile),
	}

	if err := s.eventBus.Publish(ctx, "TRANSCODE_EVENTS", event); err != nil {
		s.logger.Error("failed to publish job created event",
			zap.Error(err),
			zap.String("job_id", job.ID().String()))
	}

	return job, nil
}

// CreateJobWithOptions creates a new transcode job with custom options
func (s *serviceImpl) CreateJobWithOptions(ctx context.Context, inputPath, outputPath string, profile Profile, opts JobOptions) (*Job, error) {
	// Validate profile is supported
	capabilities := s.transcoder.GetCapabilities()
	supported := false
	for _, p := range capabilities.SupportedProfiles {
		if p == profile {
			supported = true
			break
		}
	}
	if !supported {
		return nil, ErrUnsupportedProfile
	}

	// Create new job with options
	job, err := NewJobWithOptions(inputPath, outputPath, profile, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Save to repository
	if err := s.repo.Save(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	// Publish event
	event := &JobCreated{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"transcode",
			"transcode.job.created",
			1,
		),
		InputPath:  inputPath,
		OutputPath: outputPath,
		Profile:    string(profile),
	}

	if err := s.eventBus.Publish(ctx, "TRANSCODE_EVENTS", event); err != nil {
		s.logger.Error("failed to publish job created event",
			zap.Error(err),
			zap.String("job_id", job.ID().String()),
		)
	}

	return job, nil
}

func (s *serviceImpl) GetJob(ctx context.Context, id uuid.UUID) (*Job, error) {
	job, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	return job, nil
}

func (s *serviceImpl) ListJobs(ctx context.Context, status *Status) ([]*Job, error) {
	var statusFilter Status
	if status != nil {
		statusFilter = *status
	}

	jobs, err := s.repo.List(ctx, statusFilter, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}

func (s *serviceImpl) StartJob(ctx context.Context, id uuid.UUID) error {
	// Get job
	job, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}

	// Start job
	if err := job.Start(); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	// Save updated job
	if err := s.repo.Save(ctx, job); err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	// Publish event
	event := &JobStarted{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"transcode",
			"transcode.job.started",
			1,
		),
	}

	if err := s.eventBus.Publish(ctx, "TRANSCODE_EVENTS", event); err != nil {
		s.logger.Error("failed to publish job started event",
			zap.Error(err),
			zap.String("job_id", job.ID().String()))
	}

	// Start transcoding in background
	go s.runTranscode(job)

	return nil
}

func (s *serviceImpl) CancelJob(ctx context.Context, id uuid.UUID) error {
	// Get job
	job, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}

	// Cancel transcoding
	if err := s.transcoder.Cancel(ctx, id); err != nil {
		s.logger.Error("failed to cancel transcoder",
			zap.Error(err),
			zap.String("job_id", id.String()))
	}

	// Update job status
	job.Cancel()

	// Save updated job
	if err := s.repo.Save(ctx, job); err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	// Publish event
	event := &JobCancelled{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"transcode",
			"transcode.job.cancelled",
			1,
		),
	}

	if err := s.eventBus.Publish(ctx, "TRANSCODE_EVENTS", event); err != nil {
		s.logger.Error("failed to publish job cancelled event",
			zap.Error(err),
			zap.String("job_id", job.ID().String()))
	}

	return nil
}

func (s *serviceImpl) RetryJob(ctx context.Context, id uuid.UUID) error {
	// Get job
	job, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}

	// Check if job can be retried
	if job.Status() != StatusFailed {
		return fmt.Errorf("can only retry failed jobs")
	}

	// Check retry count
	if !job.IncrementRetry() {
		return fmt.Errorf("job has exceeded maximum retries")
	}

	// Reset job status to pending
	// We need to create a new job instance since we can't modify private fields
	newJob := NewJobFromRepository(
		job.ID(),
		job.InputPath(),
		job.OutputPath(),
		job.Profile(),
		StatusPending,
		job.Options(),
		job.Metadata(),
		job.CreatedAt(),
		time.Now(),
	)

	// Save updated job
	if err := s.repo.Save(ctx, newJob); err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	// Start the job
	return s.StartJob(ctx, id)
}

func (s *serviceImpl) DeleteJob(ctx context.Context, id uuid.UUID) error {
	// Get job
	job, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find job: %w", err)
	}

	// Cancel if running
	if job.Status() == StatusRunning {
		if err := s.transcoder.Cancel(ctx, id); err != nil {
			s.logger.Error("failed to cancel transcoder",
				zap.Error(err),
				zap.String("job_id", id.String()))
		}
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	// TODO: Delete output files from storage

	return nil
}

func (s *serviceImpl) GetProgress(ctx context.Context, id uuid.UUID) (*Progress, error) {
	job, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find job: %w", err)
	}

	progress := job.Progress()
	return &progress, nil
}

func (s *serviceImpl) GetVariants(ctx context.Context, id uuid.UUID) ([]Variant, error) {
	job, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find job: %w", err)
	}

	if job.Status() != StatusCompleted {
		return nil, fmt.Errorf("job is not completed")
	}

	// TODO: Parse variants from output metadata
	return []Variant{}, nil
}

func (s *serviceImpl) runTranscode(job *Job) {
	ctx := context.Background()
	progressChan := make(chan Progress, 10)

	// Monitor progress
	go s.monitorProgress(ctx, job.ID(), progressChan)

	// Run transcode
	err := s.transcoder.Transcode(ctx, job, progressChan)
	close(progressChan)

	// Update job status
	if err != nil {
		if err == ErrJobCancelled {
			// Already handled by CancelJob
			return
		}

		job.Fail(err.Error())
		s.logger.Error("transcode failed",
			zap.Error(err),
			zap.String("job_id", job.ID().String()))

		// Publish failure event
		event := &JobFailed{
			BaseEvent: domainevents.NewBaseEvent(
				job.ID(),
				"transcode",
				"transcode.job.failed",
				1,
			),
			Error:      err.Error(),
			RetryCount: job.RetryCount(),
			CanRetry:   job.RetryCount() < job.MaxRetries(),
		}

		if err := s.eventBus.Publish(ctx, "TRANSCODE_EVENTS", event); err != nil {
			s.logger.Error("failed to publish job failed event",
				zap.Error(err),
				zap.String("job_id", job.ID().String()))
		}
	} else {
		if err := job.Complete(); err != nil {
			s.logger.Error("failed to complete job",
				zap.Error(err),
				zap.String("job_id", job.ID().String()))
			return
		}

		// Calculate duration
		duration := time.Since(*job.StartedAt())

		// TODO: Get actual variant count and size from output
		event := &JobCompleted{
			BaseEvent: domainevents.NewBaseEvent(
				job.ID(),
				"transcode",
				"transcode.job.completed",
				1,
			),
			OutputPath:   job.OutputPath(),
			Duration:     duration,
			VariantCount: 4, // TODO: Get actual count
			TotalSize:    0, // TODO: Get actual size
		}

		if err := s.eventBus.Publish(ctx, "TRANSCODE_EVENTS", event); err != nil {
			s.logger.Error("failed to publish job completed event",
				zap.Error(err),
				zap.String("job_id", job.ID().String()))
		}
	}

	// Save final job state
	if err := s.repo.Save(ctx, job); err != nil {
		s.logger.Error("failed to save final job state",
			zap.Error(err),
			zap.String("job_id", job.ID().String()))
	}
}

func (s *serviceImpl) monitorProgress(ctx context.Context, jobID uuid.UUID, progressChan <-chan Progress) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastProgress Progress
	progressUpdated := false

	for {
		select {
		case <-ctx.Done():
			return

		case progress, ok := <-progressChan:
			if !ok {
				return
			}
			lastProgress = progress
			progressUpdated = true

		case <-ticker.C:
			if progressUpdated {
				progressUpdated = false

				// Get latest job state
				job, err := s.repo.FindByID(ctx, jobID)
				if err != nil {
					s.logger.Error("failed to get job for progress update",
						zap.Error(err),
						zap.String("job_id", jobID.String()))
					continue
				}

				// Update progress
				job.UpdateProgress(lastProgress)

				// Save job
				if err := s.repo.Save(ctx, job); err != nil {
					s.logger.Error("failed to save job progress",
						zap.Error(err),
						zap.String("job_id", jobID.String()))
				}

				// Publish progress event
				event := &JobProgress{
					BaseEvent: domainevents.NewBaseEvent(
						jobID,
						"transcode",
						"transcode.job.progress",
						1,
					),
					Percent:     lastProgress.Percent,
					CurrentTime: lastProgress.CurrentTime,
					Speed:       lastProgress.Speed,
					ETA:         lastProgress.ETA,
				}

				if err := s.eventBus.Publish(ctx, "TRANSCODE_EVENTS", event); err != nil {
					s.logger.Error("failed to publish progress event",
						zap.Error(err),
						zap.String("job_id", jobID.String()))
				}
			}
		}
	}
}