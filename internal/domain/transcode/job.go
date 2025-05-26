package transcode

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
)

// Status represents the status of a transcode job
type Status string

const (
	StatusPending     Status = "pending"
	StatusRunning     Status = "running"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
)

// Profile represents a transcode profile
type Profile string

const (
	ProfileHLS      Profile = "hls"       // Multi-bitrate HLS
	ProfileHLS1080p Profile = "hls_1080p"
	ProfileHLS720p  Profile = "hls_720p"
	ProfileHLS480p  Profile = "hls_480p"
	ProfileMP4      Profile = "mp4"
	ProfileWebM     Profile = "webm"
	ProfileCustom   Profile = "custom"
)

// Job represents a transcode job
type Job struct {
	id           uuid.UUID
	inputPath    string
	outputPath   string
	profile      Profile
	status       Status
	progress     Progress
	options      Options
	metadata     Metadata
	startedAt    *time.Time
	completedAt  *time.Time
	error        string
	retryCount   int
	maxRetries   int
	createdAt    time.Time
	updatedAt    time.Time
	events       []domainevents.Event // For event sourcing
}

// Progress tracks transcode progress
type Progress struct {
	Percent       float64
	CurrentTime   time.Duration // Current position in the video
	TotalDuration time.Duration // Total video duration
	FPS           float64       // Current encoding FPS
	Bitrate       int64         // Current bitrate
	Speed         float64       // Encoding speed (e.g., 1.5x)
	ETA           time.Duration
}

// JobOptions contains options for creating a job
type JobOptions struct {
	VideoCodec    string
	AudioCodec    string
	Container     string
	Bitrate       int    // in kbps
	Width         int
	Height        int
	FrameRate     float32
}

// Options contains transcode options
type Options struct {
	VideoCodec    string            // e.g., "libx264"
	AudioCodec    string            // e.g., "aac"
	Preset        string            // e.g., "fast", "medium", "slow"
	CRF           int               // Constant Rate Factor (quality)
	MaxBitrate    string            // e.g., "5M"
	BufferSize    string            // e.g., "10M"
	AudioBitrate  string            // e.g., "128k"
	Resolution    string            // e.g., "1920x1080"
	FrameRate     int               // e.g., 30
	SegmentTime   int               // HLS segment duration in seconds
	PlaylistType  string            // "vod" or "event"
	CustomFlags   map[string]string // Additional FFmpeg flags
}

// Metadata contains video metadata
type Metadata struct {
	Duration      time.Duration
	VideoCodec    string
	AudioCodec    string
	Width         int
	Height        int
	FrameRate     float64
	Bitrate       int64
	HasAudio      bool
	AudioChannels int
	Container     string
}

// Variant represents an HLS variant
type Variant struct {
	Resolution string
	Bitrate    string
	Bandwidth  int64
	PlaylistPath string
}

// NewJob creates a new transcode job
func NewJob(inputPath, outputPath string, profile Profile) (*Job, error) {
	if inputPath == "" {
		return nil, fmt.Errorf("input path is required")
	}
	if outputPath == "" {
		return nil, fmt.Errorf("output path is required")
	}

	now := time.Now()
	return &Job{
		id:         uuid.New(),
		inputPath:  inputPath,
		outputPath: outputPath,
		profile:    profile,
		status:     StatusPending,
		progress:   Progress{},
		options:    DefaultOptions(profile),
		maxRetries: 3,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

// NewJobWithOptions creates a new transcode job with custom options
func NewJobWithOptions(inputPath, outputPath string, profile Profile, opts JobOptions) (*Job, error) {
	if inputPath == "" {
		return nil, fmt.Errorf("input path is required")
	}
	if outputPath == "" {
		return nil, fmt.Errorf("output path is required")
	}

	now := time.Now()
	
	// Start with default options for the profile
	options := DefaultOptions(profile)
	
	// Override with custom options
	if opts.VideoCodec != "" {
		options.VideoCodec = opts.VideoCodec
	}
	if opts.AudioCodec != "" {
		options.AudioCodec = opts.AudioCodec
	}
	if opts.Width > 0 && opts.Height > 0 {
		options.Resolution = fmt.Sprintf("%dx%d", opts.Width, opts.Height)
	}
	if opts.Bitrate > 0 {
		options.MaxBitrate = fmt.Sprintf("%dk", opts.Bitrate)
		options.BufferSize = fmt.Sprintf("%dk", opts.Bitrate*2)
	}
	
	job := &Job{
		id:         uuid.New(),
		inputPath:  inputPath,
		outputPath: outputPath,
		profile:    profile,
		status:     StatusPending,
		progress:   Progress{},
		options:    options,
		maxRetries: 3,
		createdAt:  now,
		updatedAt:  now,
	}
	
	// Set metadata if provided
	if opts.Container != "" {
		job.metadata.Container = opts.Container
	}
	
	return job, nil
}

// DefaultOptions returns default options for a profile
func DefaultOptions(profile Profile) Options {
	base := Options{
		VideoCodec:   "libx264",
		AudioCodec:   "aac",
		Preset:       "fast",
		CRF:          23,
		AudioBitrate: "128k",
		SegmentTime:  6,
		PlaylistType: "vod",
		CustomFlags:  make(map[string]string),
	}

	switch profile {
	case ProfileHLS1080p:
		base.Resolution = "1920x1080"
		base.MaxBitrate = "5M"
		base.BufferSize = "10M"
	case ProfileHLS720p:
		base.Resolution = "1280x720"
		base.MaxBitrate = "3M"
		base.BufferSize = "6M"
	case ProfileHLS480p:
		base.Resolution = "854x480"
		base.MaxBitrate = "1.5M"
		base.BufferSize = "3M"
	case ProfileHLS:
		// Multi-bitrate HLS profile will generate multiple resolutions
		base.Resolution = "auto"
	}

	return base
}

// Getters
func (j *Job) ID() uuid.UUID           { return j.id }
func (j *Job) InputPath() string       { return j.inputPath }
func (j *Job) OutputPath() string      { return j.outputPath }
func (j *Job) Profile() Profile        { return j.profile }
func (j *Job) Status() Status          { return j.status }
func (j *Job) Progress() Progress      { return j.progress }
func (j *Job) Options() Options        { return j.options }
func (j *Job) Metadata() Metadata      { return j.metadata }
func (j *Job) StartedAt() *time.Time   { return j.startedAt }
func (j *Job) CompletedAt() *time.Time { return j.completedAt }
func (j *Job) Error() string           { return j.error }
func (j *Job) RetryCount() int         { return j.retryCount }
func (j *Job) MaxRetries() int         { return j.maxRetries }
func (j *Job) CreatedAt() time.Time    { return j.createdAt }
func (j *Job) UpdatedAt() time.Time    { return j.updatedAt }
func (j *Job) Events() []domainevents.Event { return j.events }

// NewJobFromRepository creates a job from repository data
func NewJobFromRepository(
	id uuid.UUID,
	inputPath string,
	outputPath string,
	profile Profile,
	status Status,
	options Options,
	metadata Metadata,
	createdAt time.Time,
	updatedAt time.Time,
) *Job {
	return &Job{
		id:         id,
		inputPath:  inputPath,
		outputPath: outputPath,
		profile:    profile,
		status:     status,
		options:    options,
		metadata:   metadata,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
		events:     []domainevents.Event{},
	}
}

// Setters for repository
func (j *Job) SetProgress(progress Progress) {
	j.progress = progress
}

func (j *Job) SetStartedAt(t time.Time) {
	j.startedAt = &t
}

func (j *Job) SetCompletedAt(t time.Time) {
	j.completedAt = &t
}

func (j *Job) SetError(err string) {
	j.error = err
}

// Start marks the job as started
func (j *Job) Start() error {
	if j.status != StatusPending {
		return fmt.Errorf("cannot start job in status %s", j.status)
	}

	now := time.Now()
	j.status = StatusRunning
	j.startedAt = &now
	j.updatedAt = now
	return nil
}

// Complete marks the job as completed
func (j *Job) Complete() error {
	if j.status != StatusRunning {
		return fmt.Errorf("cannot complete job in status %s", j.status)
	}

	now := time.Now()
	j.status = StatusCompleted
	j.completedAt = &now
	j.updatedAt = now
	return nil
}

// Fail marks the job as failed
func (j *Job) Fail(err string) {
	j.status = StatusFailed
	j.error = err
	j.updatedAt = time.Now()
}

// Cancel cancels the job
func (j *Job) Cancel() {
	j.status = StatusCancelled
	j.updatedAt = time.Now()
}

// UpdateProgress updates job progress
func (j *Job) UpdateProgress(progress Progress) {
	j.progress = progress
	j.updatedAt = time.Now()
}

// SetOptions sets transcode options
func (j *Job) SetOptions(options Options) {
	j.options = options
	j.updatedAt = time.Now()
}

// SetMetadata sets video metadata
func (j *Job) SetMetadata(metadata Metadata) {
	j.metadata = metadata
	j.updatedAt = time.Now()
}

// IncrementRetry increments the retry count
func (j *Job) IncrementRetry() bool {
	j.retryCount++
	j.updatedAt = time.Now()
	return j.retryCount <= j.maxRetries
}

// CalculateETA calculates estimated time of arrival
func (p *Progress) CalculateETA() time.Duration {
	if p.Percent <= 0 || p.Percent >= 100 {
		return 0
	}

	if p.Speed <= 0 {
		return 0
	}

	remainingDuration := p.TotalDuration - p.CurrentTime
	eta := time.Duration(float64(remainingDuration) / p.Speed)

	return eta
}