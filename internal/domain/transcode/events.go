package transcode

import (
	"time"

	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
)

// JobCreated is emitted when a transcode job is created
type JobCreated struct {
	domainevents.BaseEvent
	InputPath  string `json:"input_path"`
	OutputPath string `json:"output_path"`
	Profile    string `json:"profile"`
}

// NewJobCreated creates a new JobCreated event
func NewJobCreated(job *Job) *JobCreated {
	return &JobCreated{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"TranscodeJob",
			"TranscodeJobCreated",
			1,
		),
		InputPath:  job.InputPath(),
		OutputPath: job.OutputPath(),
		Profile:    string(job.Profile()),
	}
}

// JobStarted is emitted when a transcode job starts
type JobStarted struct {
	domainevents.BaseEvent
	InputPath string `json:"input_path"`
}

// NewJobStarted creates a new JobStarted event
func NewJobStarted(job *Job) *JobStarted {
	return &JobStarted{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"TranscodeJob",
			"TranscodeJobStarted",
			1,
		),
		InputPath: job.InputPath(),
	}
}

// JobProgress is emitted periodically with transcode progress
type JobProgress struct {
	domainevents.BaseEvent
	Percent       float64       `json:"percent"`
	CurrentTime   time.Duration `json:"current_time"`
	TotalDuration time.Duration `json:"total_duration"`
	FPS           float64       `json:"fps"`
	Speed         float64       `json:"speed"`
	ETA           time.Duration `json:"eta"`
}

// NewJobProgress creates a new JobProgress event
func NewJobProgress(job *Job) *JobProgress {
	progress := job.Progress()
	return &JobProgress{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"TranscodeJob",
			"TranscodeJobProgress",
			1,
		),
		Percent:       progress.Percent,
		CurrentTime:   progress.CurrentTime,
		TotalDuration: progress.TotalDuration,
		FPS:           progress.FPS,
		Speed:         progress.Speed,
		ETA:           progress.ETA,
	}
}

// JobCompleted is emitted when a transcode job completes
type JobCompleted struct {
	domainevents.BaseEvent
	OutputPath    string        `json:"output_path"`
	Duration      time.Duration `json:"duration"`
	VariantCount  int           `json:"variant_count"`
	TotalSize     int64         `json:"total_size"`
}

// NewJobCompleted creates a new JobCompleted event
func NewJobCompleted(job *Job, variantCount int, totalSize int64) *JobCompleted {
	var duration time.Duration
	if job.StartedAt() != nil && job.CompletedAt() != nil {
		duration = job.CompletedAt().Sub(*job.StartedAt())
	}

	return &JobCompleted{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"TranscodeJob",
			"TranscodeJobCompleted",
			1,
		),
		OutputPath:   job.OutputPath(),
		Duration:     duration,
		VariantCount: variantCount,
		TotalSize:    totalSize,
	}
}

// JobFailed is emitted when a transcode job fails
type JobFailed struct {
	domainevents.BaseEvent
	Error      string `json:"error"`
	RetryCount int    `json:"retry_count"`
	CanRetry   bool   `json:"can_retry"`
}

// NewJobFailed creates a new JobFailed event
func NewJobFailed(job *Job, canRetry bool) *JobFailed {
	return &JobFailed{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"TranscodeJob",
			"TranscodeJobFailed",
			1,
		),
		Error:      job.Error(),
		RetryCount: job.RetryCount(),
		CanRetry:   canRetry,
	}
}

// JobCancelled is emitted when a transcode job is cancelled
type JobCancelled struct {
	domainevents.BaseEvent
}

// NewJobCancelled creates a new JobCancelled event
func NewJobCancelled(job *Job) *JobCancelled {
	return &JobCancelled{
		BaseEvent: domainevents.NewBaseEvent(
			job.ID(),
			"TranscodeJob",
			"TranscodeJobCancelled",
			1,
		),
	}
}