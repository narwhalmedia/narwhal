package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/domain/download"
	domainevents "github.com/narwhalmedia/narwhal/internal/domain/events"
)

// serviceImpl implements the download service
type serviceImpl struct {
	repository      download.Repository
	eventPublisher  EventPublisher
	httpDownloader  *HTTPDownloader
	torrentManager  *TorrentManager
	validator       *FileValidator
	activeDownloads map[uuid.UUID]*activeDownload
	mu              sync.RWMutex
	logger          *zap.Logger
	downloadDir     string
}

// EventPublisher interface for publishing events
type EventPublisher interface {
	PublishEvent(ctx context.Context, event domainevents.Event) error
}

// activeDownload tracks an active download
type activeDownload struct {
	download *download.Download
	cancel   context.CancelFunc
	progress chan download.Progress
}

// NewService creates a new download service
func NewService(
	repository download.Repository,
	eventPublisher EventPublisher,
	downloadDir string,
	torrentDataDir string,
	logger *zap.Logger,
) (download.Service, error) {
	// Create download directory if it doesn't exist
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Create torrent manager
	torrentManager, err := NewTorrentManager(torrentDataDir, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent manager: %w", err)
	}

	return &serviceImpl{
		repository:      repository,
		eventPublisher:  eventPublisher,
		httpDownloader:  NewHTTPDownloader(logger),
		torrentManager:  torrentManager,
		validator:       NewFileValidator(logger),
		activeDownloads: make(map[uuid.UUID]*activeDownload),
		logger:          logger.Named("download-service"),
		downloadDir:     downloadDir,
	}, nil
}

// CreateDownload creates a new download
func (s *serviceImpl) CreateDownload(ctx context.Context, url string, downloadType download.Type, targetPath string) (*download.Download, error) {
	// Create download entity
	dl, err := download.NewDownload(url, downloadType, targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create download: %w", err)
	}

	// Save to repository
	if err := s.repository.Save(ctx, dl); err != nil {
		return nil, fmt.Errorf("failed to save download: %w", err)
	}

	// Publish event
	event := download.NewDownloadCreated(dl)
	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		s.logger.Error("failed to publish download created event", zap.Error(err))
	}

	s.logger.Info("download created",
		zap.String("id", dl.ID().String()),
		zap.String("url", dl.URL()),
		zap.String("type", string(dl.Type())),
	)

	return dl, nil
}

// GetDownload retrieves a download by ID
func (s *serviceImpl) GetDownload(ctx context.Context, id uuid.UUID) (*download.Download, error) {
	return s.repository.FindByID(ctx, id)
}

// ListDownloads lists all downloads with optional filtering
func (s *serviceImpl) ListDownloads(ctx context.Context, status *download.Status) ([]*download.Download, error) {
	if status != nil {
		return s.repository.FindByStatus(ctx, *status)
	}
	return s.repository.FindAll(ctx)
}

// StartDownload starts a download
func (s *serviceImpl) StartDownload(ctx context.Context, id uuid.UUID) error {
	// Get download
	dl, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find download: %w", err)
	}

	// Start download
	if err := dl.Start(); err != nil {
		return err
	}

	// Save updated status
	if err := s.repository.Save(ctx, dl); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	// Publish event
	event := download.NewDownloadStarted(dl)
	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		s.logger.Error("failed to publish download started event", zap.Error(err))
	}

	// Start download in background
	s.startDownloadWorker(dl)

	return nil
}

// PauseDownload pauses a download
func (s *serviceImpl) PauseDownload(ctx context.Context, id uuid.UUID) error {
	// Get download
	dl, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find download: %w", err)
	}

	// Pause download
	if err := dl.Pause(); err != nil {
		return err
	}

	// Cancel active download
	s.mu.Lock()
	if active, ok := s.activeDownloads[id]; ok {
		active.cancel()
		delete(s.activeDownloads, id)
	}
	s.mu.Unlock()

	// Save updated status
	if err := s.repository.Save(ctx, dl); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	// Publish event
	event := download.NewDownloadPaused(dl)
	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		s.logger.Error("failed to publish download paused event", zap.Error(err))
	}

	// Pause torrent if applicable
	if dl.Type() == download.TypeTorrent {
		if err := s.torrentManager.PauseDownload(dl.ID().String()); err != nil {
			s.logger.Error("failed to pause torrent", zap.Error(err))
		}
	}

	return nil
}

// ResumeDownload resumes a paused download
func (s *serviceImpl) ResumeDownload(ctx context.Context, id uuid.UUID) error {
	// Get download
	dl, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find download: %w", err)
	}

	// Resume download
	if err := dl.Resume(); err != nil {
		return err
	}

	// Save updated status
	if err := s.repository.Save(ctx, dl); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	// Publish event
	event := download.NewDownloadResumed(dl)
	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		s.logger.Error("failed to publish download resumed event", zap.Error(err))
	}

	// Resume download in background
	s.startDownloadWorker(dl)

	return nil
}

// CancelDownload cancels a download
func (s *serviceImpl) CancelDownload(ctx context.Context, id uuid.UUID) error {
	// Get download
	dl, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find download: %w", err)
	}

	// Cancel download
	dl.Cancel()

	// Cancel active download
	s.mu.Lock()
	if active, ok := s.activeDownloads[id]; ok {
		active.cancel()
		delete(s.activeDownloads, id)
	}
	s.mu.Unlock()

	// Save updated status
	if err := s.repository.Save(ctx, dl); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	// Publish event
	event := download.NewDownloadCancelled(dl)
	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		s.logger.Error("failed to publish download cancelled event", zap.Error(err))
	}

	// Cancel torrent if applicable
	if dl.Type() == download.TypeTorrent {
		if err := s.torrentManager.CancelDownload(dl.ID().String()); err != nil {
			s.logger.Error("failed to cancel torrent", zap.Error(err))
		}
	}

	// Clean up partial file if exists
	if err := os.Remove(dl.TargetPath()); err != nil && !os.IsNotExist(err) {
		s.logger.Error("failed to remove partial file", zap.Error(err))
	}

	return nil
}

// RetryDownload retries a failed download
func (s *serviceImpl) RetryDownload(ctx context.Context, id uuid.UUID) error {
	// Get download
	dl, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find download: %w", err)
	}

	// Check if retry is allowed
	if !dl.IncrementRetry() {
		return fmt.Errorf("maximum retries exceeded")
	}

	// Reset to pending
	newDl, err := download.NewDownload(dl.URL(), dl.Type(), dl.TargetPath())
	if err != nil {
		return err
	}

	// Copy metadata
	newDl.SetMetadata(dl.Metadata())
	newDl.SetChecksum(dl.Checksum(), dl.ChecksumType())

	// Start download
	if err := newDl.Start(); err != nil {
		return err
	}

	// Save updated download
	if err := s.repository.Save(ctx, newDl); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	// Start download in background
	s.startDownloadWorker(newDl)

	return nil
}

// DeleteDownload deletes a download record
func (s *serviceImpl) DeleteDownload(ctx context.Context, id uuid.UUID) error {
	// Cancel if active
	s.mu.Lock()
	if active, ok := s.activeDownloads[id]; ok {
		active.cancel()
		delete(s.activeDownloads, id)
	}
	s.mu.Unlock()

	return s.repository.Delete(ctx, id)
}

// GetProgress gets current progress for a download
func (s *serviceImpl) GetProgress(ctx context.Context, id uuid.UUID) (*download.Progress, error) {
	// Check active downloads first
	s.mu.RLock()
	active, ok := s.activeDownloads[id]
	s.mu.RUnlock()

	if ok {
		select {
		case progress := <-active.progress:
			return &progress, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Return last known progress from download
		}
	}

	// Get from repository
	dl, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	progress := dl.Progress()
	return &progress, nil
}

// startDownloadWorker starts a download worker in the background
func (s *serviceImpl) startDownloadWorker(dl *download.Download) {
	ctx, cancel := context.WithCancel(context.Background())
	progressChan := make(chan download.Progress, 10)

	// Register active download
	s.mu.Lock()
	s.activeDownloads[dl.ID()] = &activeDownload{
		download: dl,
		cancel:   cancel,
		progress: progressChan,
	}
	s.mu.Unlock()

	// Start worker
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.activeDownloads, dl.ID())
			s.mu.Unlock()
			close(progressChan)
		}()

		// Run download
		err := s.runDownload(ctx, dl, progressChan)
		if err != nil {
			s.logger.Error("download failed",
				zap.String("id", dl.ID().String()),
				zap.Error(err),
			)
			dl.Fail(err.Error())
			
			// Publish failure event
			event := download.NewDownloadFailed(dl, dl.RetryCount() < dl.MaxRetries())
			if pubErr := s.eventPublisher.PublishEvent(context.Background(), event); pubErr != nil {
				s.logger.Error("failed to publish download failed event", zap.Error(pubErr))
			}
		} else {
			// Mark as completed
			if err := dl.Complete(); err != nil {
				s.logger.Error("failed to mark download as completed", zap.Error(err))
			}
			
			// Publish completion event
			event := download.NewDownloadCompleted(dl)
			if pubErr := s.eventPublisher.PublishEvent(context.Background(), event); pubErr != nil {
				s.logger.Error("failed to publish download completed event", zap.Error(pubErr))
			}
		}

		// Save final state
		if err := s.repository.Save(context.Background(), dl); err != nil {
			s.logger.Error("failed to save download state", zap.Error(err))
		}
	}()
}

// runDownload performs the actual download
func (s *serviceImpl) runDownload(ctx context.Context, dl *download.Download, progressChan chan<- download.Progress) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(dl.TargetPath())
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create progress reporter
	progressReporter := func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case progress := <-progressChan:
				// Update download progress
				dl.UpdateProgress(progress)
				
				// Save to repository
				if err := s.repository.Save(context.Background(), dl); err != nil {
					s.logger.Error("failed to save progress", zap.Error(err))
				}
				
				// Publish progress event periodically
				event := download.NewDownloadProgress(dl)
				if err := s.eventPublisher.PublishEvent(context.Background(), event); err != nil {
					s.logger.Error("failed to publish progress event", zap.Error(err))
				}
			}
		}
	}

	// Start progress reporter
	go progressReporter()

	// Perform download based on type
	switch dl.Type() {
	case download.TypeHTTP:
		return s.runHTTPDownload(ctx, dl, progressChan)
	case download.TypeTorrent:
		return s.runTorrentDownload(ctx, dl, progressChan)
	case download.TypeUsenet:
		return fmt.Errorf("usenet downloads not yet implemented")
	default:
		return fmt.Errorf("unsupported download type: %s", dl.Type())
	}
}

// runHTTPDownload performs an HTTP download
func (s *serviceImpl) runHTTPDownload(ctx context.Context, dl *download.Download, progressChan chan<- download.Progress) error {
	// Open or create file
	file, err := os.OpenFile(dl.TargetPath(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get current file size for resume
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	offset := stat.Size()

	// Download with resume if file exists
	if offset > 0 {
		s.logger.Info("resuming download",
			zap.String("id", dl.ID().String()),
			zap.Int64("offset", offset),
		)
		err = s.httpDownloader.Resume(ctx, dl.URL(), file, offset, progressChan)
	} else {
		err = s.httpDownloader.Download(ctx, dl.URL(), file, progressChan)
	}

	if err != nil {
		return err
	}

	// Validate checksum if provided
	if dl.Checksum() != "" && dl.ChecksumType() != "" {
		if err := dl.StartVerifying(); err == nil {
			s.repository.Save(context.Background(), dl)
			
			if err := s.validator.ValidateChecksum(dl.TargetPath(), dl.Checksum(), dl.ChecksumType()); err != nil {
				return fmt.Errorf("checksum validation failed: %w", err)
			}
		}
	}

	return nil
}

// runTorrentDownload performs a torrent download
func (s *serviceImpl) runTorrentDownload(ctx context.Context, dl *download.Download, progressChan chan<- download.Progress) error {
	return s.torrentManager.StartDownload(ctx, dl.ID().String(), dl.URL(), progressChan)
}