package sagas

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/domain/download"
	"github.com/narwhalmedia/narwhal/internal/domain/media"
	"github.com/narwhalmedia/narwhal/internal/infrastructure/events/nats"
)

// MediaProcessingSaga orchestrates the download -> transcode -> ready workflow
type MediaProcessingSaga struct {
	mediaService     media.Service
	downloadService  DownloadService  // Interface to be implemented
	transcodeService TranscodeService // Interface to be implemented
	logger           *zap.Logger
}

// DownloadService interface wraps the domain download service
type DownloadService interface {
	CreateDownload(ctx context.Context, url string, downloadType download.Type, targetPath string) (*download.Download, error)
	StartDownload(ctx context.Context, id uuid.UUID) error
	CancelDownload(ctx context.Context, id uuid.UUID) error
	GetDownload(ctx context.Context, id uuid.UUID) (*download.Download, error)
}

// TranscodeService interface (to be implemented by transcode service)
type TranscodeService interface {
	StartTranscode(ctx context.Context, inputPath string, outputPath string, profile string) (string, error)
	CancelTranscode(ctx context.Context, jobID string) error
}

// NewMediaProcessingSaga creates a new media processing saga
func NewMediaProcessingSaga(
	mediaService media.Service,
	downloadService DownloadService,
	transcodeService TranscodeService,
	logger *zap.Logger,
) *MediaProcessingSaga {
	return &MediaProcessingSaga{
		mediaService:     mediaService,
		downloadService:  downloadService,
		transcodeService: transcodeService,
		logger:           logger.Named("media-processing-saga"),
	}
}

// GetDefinition returns the saga definition
func (s *MediaProcessingSaga) GetDefinition() *nats.SagaDefinition {
	return &nats.SagaDefinition{
		Type: "MediaProcessing",
		Steps: []nats.SagaStep{
			&UpdateMediaStatusStep{saga: s, status: media.StatusDownloading},
			&DownloadMediaStep{saga: s},
			&UpdateMediaStatusStep{saga: s, status: media.StatusTranscoding},
			&TranscodeMediaStep{saga: s},
			&UpdateMediaStatusStep{saga: s, status: media.StatusReady},
		},
	}
}

// Step 1: Update media status to downloading
type UpdateMediaStatusStep struct {
	saga   *MediaProcessingSaga
	status media.Status
}

func (s *UpdateMediaStatusStep) Name() string {
	return fmt.Sprintf("UpdateMediaStatus_%s", s.status)
}

func (s *UpdateMediaStatusStep) Execute(ctx context.Context, saga *nats.Saga) error {
	mediaID, ok := saga.Data["media_id"].(string)
	if !ok {
		return fmt.Errorf("media_id not found in saga data")
	}

	mediaType, ok := saga.Data["media_type"].(string)
	if !ok {
		return fmt.Errorf("media_type not found in saga data")
	}

	id, err := uuid.Parse(mediaID)
	if err != nil {
		return fmt.Errorf("invalid media_id: %w", err)
	}

	// Update status based on media type
	switch mediaType {
	case "movie":
		return s.saga.mediaService.UpdateMovieStatus(ctx, id, s.status)
	case "episode":
		return s.saga.mediaService.UpdateEpisodeStatus(ctx, id, s.status)
	default:
		return fmt.Errorf("unknown media type: %s", mediaType)
	}
}

func (s *UpdateMediaStatusStep) Compensate(ctx context.Context, saga *nats.Saga) error {
	// Revert to error status
	mediaID, _ := saga.Data["media_id"].(string)
	mediaType, _ := saga.Data["media_type"].(string)
	
	if mediaID == "" || mediaType == "" {
		return nil
	}

	id, err := uuid.Parse(mediaID)
	if err != nil {
		return nil
	}

	switch mediaType {
	case "movie":
		return s.saga.mediaService.UpdateMovieStatus(ctx, id, media.StatusError)
	case "episode":
		return s.saga.mediaService.UpdateEpisodeStatus(ctx, id, media.StatusError)
	}

	return nil
}

// Step 2: Download media
type DownloadMediaStep struct {
	saga *MediaProcessingSaga
}

func (s *DownloadMediaStep) Name() string {
	return "DownloadMedia"
}

func (s *DownloadMediaStep) Execute(ctx context.Context, saga *nats.Saga) error {
	url, ok := saga.Data["download_url"].(string)
	if !ok {
		return fmt.Errorf("download_url not found in saga data")
	}

	targetPath, ok := saga.Data["target_path"].(string)
	if !ok {
		return fmt.Errorf("target_path not found in saga data")
	}

	// Determine download type from URL
	downloadType := download.TypeHTTP
	if len(url) >= 8 && url[:8] == "magnet:?" {
		downloadType = download.TypeTorrent
	} else if len(url) > 4 && url[len(url)-4:] == ".nzb" {
		downloadType = download.TypeUsenet
	}

	// Create download
	dl, err := s.saga.downloadService.CreateDownload(ctx, url, downloadType, targetPath)
	if err != nil {
		return fmt.Errorf("failed to create download: %w", err)
	}

	// Start download
	if err := s.saga.downloadService.StartDownload(ctx, dl.ID()); err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}

	// Store download ID for potential compensation
	saga.Data["download_id"] = dl.ID().String()
	saga.Data["downloaded_path"] = targetPath

	s.saga.logger.Info("download started",
		zap.String("saga_id", saga.ID),
		zap.String("download_id", dl.ID().String()),
		zap.String("url", url),
		zap.String("type", string(downloadType)),
	)

	// TODO: Wait for download completion event
	// For now, we'll assume the download completes successfully
	// In production, this would subscribe to download events

	return nil
}

func (s *DownloadMediaStep) Compensate(ctx context.Context, saga *nats.Saga) error {
	downloadID, ok := saga.Data["download_id"].(string)
	if !ok {
		return nil
	}

	id, err := uuid.Parse(downloadID)
	if err != nil {
		s.saga.logger.Error("invalid download ID",
			zap.Error(err),
			zap.String("download_id", downloadID),
		)
		return nil
	}

	// Cancel download if still in progress
	if err := s.saga.downloadService.CancelDownload(ctx, id); err != nil {
		s.saga.logger.Error("failed to cancel download",
			zap.Error(err),
			zap.String("download_id", downloadID),
		)
	}

	// File cleanup is handled by the download service

	return nil
}

// Step 3: Transcode media
type TranscodeMediaStep struct {
	saga *MediaProcessingSaga
}

func (s *TranscodeMediaStep) Name() string {
	return "TranscodeMedia"
}

func (s *TranscodeMediaStep) Execute(ctx context.Context, saga *nats.Saga) error {
	inputPath, ok := saga.Data["downloaded_path"].(string)
	if !ok {
		return fmt.Errorf("downloaded_path not found in saga data")
	}

	outputPath, ok := saga.Data["output_path"].(string)
	if !ok {
		return fmt.Errorf("output_path not found in saga data")
	}

	profile, ok := saga.Data["transcode_profile"].(string)
	if !ok {
		profile = "default" // Default HLS profile
	}

	// Start transcode
	jobID, err := s.saga.transcodeService.StartTranscode(ctx, inputPath, outputPath, profile)
	if err != nil {
		return fmt.Errorf("failed to start transcode: %w", err)
	}

	// Store job ID for potential compensation
	saga.Data["transcode_job_id"] = jobID
	saga.Data["transcoded_path"] = outputPath

	// Update media file path
	mediaID, _ := saga.Data["media_id"].(string)
	mediaType, _ := saga.Data["media_type"].(string)
	
	if mediaID != "" && mediaType != "" {
		id, _ := uuid.Parse(mediaID)
		switch mediaType {
		case "movie":
			s.saga.mediaService.UpdateMovieFile(ctx, id, outputPath)
		case "episode":
			s.saga.mediaService.UpdateEpisodeFile(ctx, id, outputPath)
		}
	}

	s.saga.logger.Info("transcode started",
		zap.String("saga_id", saga.ID),
		zap.String("job_id", jobID),
		zap.String("input", inputPath),
		zap.String("output", outputPath),
	)

	return nil
}

func (s *TranscodeMediaStep) Compensate(ctx context.Context, saga *nats.Saga) error {
	jobID, ok := saga.Data["transcode_job_id"].(string)
	if !ok {
		return nil
	}

	// Cancel transcode if still in progress
	if err := s.saga.transcodeService.CancelTranscode(ctx, jobID); err != nil {
		s.saga.logger.Error("failed to cancel transcode",
			zap.Error(err),
			zap.String("job_id", jobID),
		)
	}

	// TODO: Clean up any output files

	return nil
}