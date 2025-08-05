package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/internal/library/repository"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// LibraryService handles library business logic
type LibraryService struct {
	repo     repository.Repository
	eventBus interfaces.EventBus
	cache    interfaces.Cache
	logger   interfaces.Logger
	scanner  *domain.Scanner
}

// NewLibraryService creates a new library service
func NewLibraryService(
	repo repository.Repository,
	eventBus interfaces.EventBus,
	cache interfaces.Cache,
	logger interfaces.Logger,
) *LibraryService {
	return &LibraryService{
		repo:     repo,
		eventBus: eventBus,
		cache:    cache,
		logger:   logger,
		scanner:  domain.NewScanner(logger),
	}
}

// CreateLibrary creates a new media library
func (s *LibraryService) CreateLibrary(ctx context.Context, library *domain.Library) error {
	// Validate input
	if library.Name == "" || library.Path == "" {
		return errors.BadRequest("library name and path are required")
	}

	// Check if path already exists
	existing, _ := s.repo.GetLibraryByPath(ctx, library.Path)
	if existing != nil {
		return errors.Conflict("library path already exists")
	}

	// Generate ID if not set
	if library.ID == uuid.Nil {
		library.ID = uuid.New()
	}

	// Create library
	if err := s.repo.CreateLibrary(ctx, library); err != nil {
		s.logger.Error("Failed to create library", interfaces.Error(err))
		return err
	}

	// Publish event
	s.eventBus.PublishAsync(ctx, domain.NewLibraryCreatedEvent(library))

	s.logger.Info("Library created",
		interfaces.String("id", library.ID.String()),
		interfaces.String("name", library.Name),
		interfaces.String("path", library.Path))

	return nil
}

// GetLibrary retrieves a library by ID
func (s *LibraryService) GetLibrary(ctx context.Context, id uuid.UUID) (*domain.Library, error) {
	// Check cache first
	cacheKey := "library:" + id.String()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		if library, ok := cached.(*domain.Library); ok {
			return library, nil
		}
	}

	// Get from repository
	library, err := s.repo.GetLibrary(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, library, 5*time.Minute)

	return library, nil
}

// ListLibraries lists all libraries
func (s *LibraryService) ListLibraries(ctx context.Context, enabled *bool) ([]*domain.Library, error) {
	return s.repo.ListLibraries(ctx, enabled)
}

// UpdateLibrary updates a library
func (s *LibraryService) UpdateLibrary(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*domain.Library, error) {
	// Get existing library
	library, err := s.repo.GetLibrary(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok && name != "" {
		library.Name = name
	}
	if path, ok := updates["path"].(string); ok && path != "" {
		library.Path = path
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		library.Enabled = enabled
	}
	if scanInterval, ok := updates["scan_interval"].(int); ok && scanInterval > 0 {
		library.ScanInterval = scanInterval
	}

	// Update in repository
	if err := s.repo.UpdateLibrary(ctx, library); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.cache.Delete(ctx, "library:"+id.String())

	// Publish event
	s.eventBus.PublishAsync(ctx, domain.NewLibraryUpdatedEvent(library))

	return library, nil
}

// DeleteLibrary deletes a library
func (s *LibraryService) DeleteLibrary(ctx context.Context, id uuid.UUID) error {
	// Check if library exists
	library, err := s.repo.GetLibrary(ctx, id)
	if err != nil {
		return err
	}

	// Delete library (cascades to media items)
	if err := s.repo.DeleteLibrary(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	s.cache.Delete(ctx, "library:"+id.String())

	// Publish event
	s.eventBus.PublishAsync(ctx, domain.NewLibraryDeletedEvent(id))

	s.logger.Info("Library deleted",
		interfaces.String("id", id.String()),
		interfaces.String("name", library.Name))

	return nil
}

// ScanLibrary starts a library scan
func (s *LibraryService) ScanLibrary(ctx context.Context, id uuid.UUID) error {
	library, err := s.repo.GetLibrary(ctx, id)
	if err != nil {
		return err
	}

	// Check if scan is already in progress
	if s.scanner.IsScanning(id.String()) {
		return errors.Conflict("scan already in progress")
	}

	// Start scan asynchronously
	go s.performScan(context.Background(), library)

	return nil
}

// performScan performs the actual library scan
func (s *LibraryService) performScan(ctx context.Context, library *domain.Library) {
	// Mark library as scanning
	s.scanner.SetScanning(library.ID.String(), true)
	defer s.scanner.SetScanning(library.ID.String(), false)

	scanResult := &domain.ScanResult{
		LibraryID: library.ID,
		StartedAt: time.Now(),
	}

	// Create scan history record
	if err := s.repo.CreateScanHistory(ctx, scanResult); err != nil {
		s.logger.Error("Failed to create scan history", interfaces.Error(err))
		return
	}

	s.logger.Info("Starting library scan",
		interfaces.String("library_id", library.ID.String()),
		interfaces.String("path", library.Path))

	// Scan for media files
	files, err := s.scanner.ScanDirectory(library.Path, library.Type)
	if err != nil {
		s.logger.Error("Library scan failed",
			interfaces.String("library_id", library.ID.String()),
			interfaces.Error(err))
		
		scanResult.CompletedAt = timePtr(time.Now())
		scanResult.ErrorMessage = err.Error()
		s.repo.UpdateScanHistory(ctx, scanResult)
		return
	}

	// Process found files
	for _, file := range files {
		existing, _ := s.repo.GetMediaByPath(ctx, file.Path)
		
		if existing != nil {
			// Update existing media if file was modified
			if file.Modified.After(existing.Modified) {
				existing.Size = file.Size
				existing.Modified = file.Modified
				existing.LastScanned = time.Now()
				
				if err := s.repo.UpdateMedia(ctx, existing); err != nil {
					s.logger.Error("Failed to update media",
						interfaces.String("path", file.Path),
						interfaces.Error(err))
					continue
				}
				scanResult.FilesUpdated++
			}
		} else {
			// Create new media entry
			media := &models.Media{
				ID:          uuid.New(),
				Title:       domain.ExtractTitle(file.Path),
				Type:        models.MediaType(library.Type),
				Path:        file.Path,
				Size:        file.Size,
				Added:       time.Now(),
				Modified:    file.Modified,
				LastScanned: time.Now(),
			}

			// Add library-specific fields
			media.LibraryID = library.ID
			media.Status = "pending"
			media.FilePath = file.Path
			media.FileSize = file.Size
			media.FileModifiedAt = &file.Modified
			
			if err := s.repo.CreateMedia(ctx, media); err != nil {
				s.logger.Error("Failed to create media",
					interfaces.String("path", file.Path),
					interfaces.Error(err))
				continue
			}
			
			// Publish media added event
			s.eventBus.PublishAsync(ctx, domain.NewMediaAddedEvent(media))
			scanResult.FilesAdded++
		}
		
		scanResult.FilesScanned++
	}

	// Update library last scan time
	now := time.Now()
	library.LastScanAt = &now
	s.repo.UpdateLibrary(ctx, library)

	// Complete scan history
	scanResult.CompletedAt = timePtr(time.Now())
	s.repo.UpdateScanHistory(ctx, scanResult)

	duration := time.Since(scanResult.StartedAt)
	s.logger.Info("Library scan completed",
		interfaces.String("library_id", library.ID.String()),
		interfaces.Int("files_scanned", scanResult.FilesScanned),
		interfaces.Int("files_added", scanResult.FilesAdded),
		interfaces.Int("files_updated", scanResult.FilesUpdated),
		interfaces.Any("duration", duration))

	// Publish scan completed event
	s.eventBus.PublishAsync(ctx, domain.NewLibraryScanCompletedEvent(library, scanResult.FilesAdded, scanResult.FilesUpdated))
}

// GetMedia retrieves a media item by ID
func (s *LibraryService) GetMedia(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	// Check cache first
	cacheKey := "media:" + id.String()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		if media, ok := cached.(*models.Media); ok {
			return media, nil
		}
	}

	// Get from repository
	media, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(ctx, cacheKey, media, 5*time.Minute)

	return media, nil
}

// SearchMedia searches for media items
func (s *LibraryService) SearchMedia(ctx context.Context, query string, mediaType *string, status *string, libraryID *uuid.UUID, limit, offset int) ([]*models.Media, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	return s.repo.SearchMedia(ctx, query, mediaType, status, libraryID, limit, offset)
}

// UpdateMedia updates a media item
func (s *LibraryService) UpdateMedia(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*models.Media, error) {
	// Get existing media
	media, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if title, ok := updates["title"].(string); ok && title != "" {
		media.Title = title
	}
	if description, ok := updates["description"].(string); ok {
		media.Description = description
	}
	if releaseDate, ok := updates["release_date"].(time.Time); ok {
		media.ReleaseDate = releaseDate
	}
	if genres, ok := updates["genres"].([]string); ok {
		media.Genres = genres
	}
	if tags, ok := updates["tags"].([]string); ok {
		media.Tags = tags
	}

	// Update in repository
	if err := s.repo.UpdateMedia(ctx, media); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.cache.Delete(ctx, "media:"+id.String())

	// Publish event
	s.eventBus.PublishAsync(ctx, domain.NewMediaUpdatedEvent(media))

	return media, nil
}

// DeleteMedia deletes a media item
func (s *LibraryService) DeleteMedia(ctx context.Context, id uuid.UUID) error {
	// Check if media exists
	media, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return err
	}

	// Delete media
	if err := s.repo.DeleteMedia(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	s.cache.Delete(ctx, "media:"+id.String())

	// Publish event
	s.eventBus.PublishAsync(ctx, domain.NewMediaDeletedEvent(id.String()))

	s.logger.Info("Media deleted",
		interfaces.String("id", id.String()),
		interfaces.String("title", media.Title))

	return nil
}

// ListMediaByLibrary lists media items in a library
func (s *LibraryService) ListMediaByLibrary(ctx context.Context, libraryID uuid.UUID, status *string, limit, offset int) ([]*models.Media, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	return s.repo.ListMediaByLibrary(ctx, libraryID, status, limit, offset)
}

// GetLatestScan gets the latest scan result for a library
func (s *LibraryService) GetLatestScan(ctx context.Context, libraryID uuid.UUID) (*domain.ScanResult, error) {
	return s.repo.GetLatestScan(ctx, libraryID)
}

// Helper function to get a pointer to time
func timePtr(t time.Time) *time.Time {
	return &t
}