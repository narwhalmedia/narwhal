package repository

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/pkg/encryption"
	pkgerrors "github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/models"
	"gorm.io/gorm"
)

// GormRepository implements the repository interfaces using GORM
type GormRepository struct {
	db        *gorm.DB
	encryptor *encryption.Encryptor
}

// NewGormRepository creates a new GORM repository
func NewGormRepository(db *gorm.DB) (*GormRepository, error) {
	// Get encryption key from environment variable
	encryptionKey := os.Getenv("NARWHAL_ENCRYPTION_KEY")
	if encryptionKey == "" {
		// Use a default key for development, but log a warning
		encryptionKey = "development-key-please-change-in-production"
	}

	encryptor, err := encryption.NewEncryptor(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &GormRepository{
		db:        db,
		encryptor: encryptor,
	}, nil
}

// CreateLibrary creates a new library
func (r *GormRepository) CreateLibrary(ctx context.Context, library *domain.Library) error {
	model := &Library{
		Name:         library.Name,
		Path:         library.Path,
		MediaType:    library.Type,
		Enabled:      library.Enabled,
		ScanInterval: library.ScanInterval,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create library: %w", err)
	}

	library.ID = model.ID
	library.CreatedAt = model.CreatedAt
	library.UpdatedAt = model.UpdatedAt
	return nil
}

// GetLibrary retrieves a library by ID
func (r *GormRepository) GetLibrary(ctx context.Context, id uuid.UUID) (*domain.Library, error) {
	var model Library
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("library not found")
		}
		return nil, fmt.Errorf("failed to get library: %w", err)
	}

	return r.toDomainLibrary(&model), nil
}

// GetLibraryByPath retrieves a library by path
func (r *GormRepository) GetLibraryByPath(ctx context.Context, path string) (*domain.Library, error) {
	var model Library
	if err := r.db.WithContext(ctx).First(&model, "path = ?", path).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("library not found")
		}
		return nil, fmt.Errorf("failed to get library by path: %w", err)
	}

	return r.toDomainLibrary(&model), nil
}

// UpdateLibrary updates a library
func (r *GormRepository) UpdateLibrary(ctx context.Context, library *domain.Library) error {
	updates := map[string]interface{}{
		"name":          library.Name,
		"path":          library.Path,
		"enabled":       library.Enabled,
		"scan_interval": library.ScanInterval,
	}

	if library.LastScanAt != nil && !library.LastScanAt.IsZero() {
		updates["last_scan_at"] = library.LastScanAt
	}

	result := r.db.WithContext(ctx).Model(&Library{}).Where("id = ?", library.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update library: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("library not found")
	}

	return nil
}

// DeleteLibrary deletes a library
func (r *GormRepository) DeleteLibrary(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&Library{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete library: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("library not found")
	}

	return nil
}

// ListLibraries lists all libraries
func (r *GormRepository) ListLibraries(ctx context.Context, enabled *bool) ([]*domain.Library, error) {
	query := r.db.WithContext(ctx)
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	var items []Library
	if err := query.Order("name").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list libraries: %w", err)
	}

	libraries := make([]*domain.Library, len(items))
	for i, model := range items {
		libraries[i] = r.toDomainLibrary(&model)
	}

	return libraries, nil
}

// CreateMedia creates a new media item
func (r *GormRepository) CreateMedia(ctx context.Context, media *models.Media) error {
	model := &MediaItem{
		LibraryID:      media.LibraryID,
		Title:          media.Title,
		MediaType:      string(media.Type),
		Status:         media.Status,
		FilePath:       media.FilePath,
		FileSize:       media.FileSize,
		FileModifiedAt: media.FileModifiedAt,
		Description:    media.Description,
		ReleaseDate:    &media.ReleaseDate,
		Runtime:        media.Duration / 60, // Convert seconds to minutes
		Genres:         media.Genres,
		Tags:           media.Tags,
		TMDBID:         media.TMDBID,
		IMDBID:         media.IMDBID,
		TVDBID:         media.TVDBID,
		VideoCodec:     media.Codec,
		AudioCodec:     "", // Not available in models.Media
		Resolution:     media.Resolution,
		Bitrate:        media.Bitrate,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create media: %w", err)
	}

	media.ID = model.ID
	media.CreatedAt = model.CreatedAt
	media.UpdatedAt = model.UpdatedAt
	return nil
}

// GetMedia retrieves a media item by ID
func (r *GormRepository) GetMedia(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	var model MediaItem
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("media not found")
		}
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	return r.toDomainMedia(&model), nil
}

// GetMediaByPath retrieves a media item by file path
func (r *GormRepository) GetMediaByPath(ctx context.Context, path string) (*models.Media, error) {
	var model MediaItem
	if err := r.db.WithContext(ctx).First(&model, "file_path = ?", path).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("media not found")
		}
		return nil, fmt.Errorf("failed to get media by path: %w", err)
	}

	return r.toDomainMedia(&model), nil
}

// SearchMedia searches for media items
func (r *GormRepository) SearchMedia(ctx context.Context, query string, mediaType *string, status *string, libraryID *uuid.UUID, limit, offset int) ([]*models.Media, error) {
	q := r.db.WithContext(ctx).Model(&MediaItem{})

	// Search in title and original title
	if query != "" {
		q = q.Where("title ILIKE ? OR original_title ILIKE ?", "%"+query+"%", "%"+query+"%")
	}

	// Filter by media type
	if mediaType != nil && *mediaType != "" {
		q = q.Where("media_type = ?", *mediaType)
	}

	// Filter by status
	if status != nil && *status != "" {
		q = q.Where("status = ?", *status)
	}

	// Filter by library
	if libraryID != nil {
		q = q.Where("library_id = ?", *libraryID)
	}

	// Count total results for pagination
	var total int64
	q.Count(&total)

	// Apply pagination and ordering
	var items []MediaItem
	if err := q.Order("title").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to search media: %w", err)
	}

	media := make([]*models.Media, len(items))
	for i := range items {
		media[i] = r.toDomainMedia(&items[i])
	}

	return media, nil
}

// UpdateMedia updates a media item
func (r *GormRepository) UpdateMedia(ctx context.Context, media *models.Media) error {
	updates := map[string]interface{}{
		"title":            media.Title,
		"status":           media.Status,
		"file_path":        media.FilePath,
		"file_size":        media.FileSize,
		"file_modified_at": media.FileModifiedAt,
		"description":      media.Description,
		"release_date":     media.ReleaseDate,
		"runtime":          media.Duration / 60, // Convert seconds to minutes
		"genres":           media.Genres,
		"tags":             media.Tags,
		"tmdb_id":          media.TMDBID,
		"imdb_id":          media.IMDBID,
		"tvdb_id":          media.TVDBID,
		"video_codec":      media.Codec,
		"resolution":       media.Resolution,
		"bitrate":          media.Bitrate,
	}

	result := r.db.WithContext(ctx).Model(&MediaItem{}).Where("id = ?", media.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update media: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("media not found")
	}

	return nil
}

// DeleteMedia deletes a media item
func (r *GormRepository) DeleteMedia(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&MediaItem{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete media: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("media not found")
	}

	return nil
}

// ListMediaByLibrary lists media items by library
func (r *GormRepository) ListMediaByLibrary(ctx context.Context, libraryID uuid.UUID, status *string, limit, offset int) ([]*models.Media, error) {
	q := r.db.WithContext(ctx).Model(&MediaItem{}).Where("library_id = ?", libraryID)

	if status != nil && *status != "" {
		q = q.Where("status = ?", *status)
	}

	var items []MediaItem
	if err := q.Order("title").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list media by library: %w", err)
	}

	media := make([]*models.Media, len(items))
	for i := range items {
		media[i] = r.toDomainMedia(&items[i])
	}

	return media, nil
}

// CreateScanHistory creates a new scan history record
func (r *GormRepository) CreateScanHistory(ctx context.Context, scan *domain.ScanResult) error {
	model := &ScanHistory{
		LibraryID:    scan.LibraryID,
		StartedAt:    scan.StartedAt,
		CompletedAt:  scan.CompletedAt,
		FilesScanned: scan.FilesScanned,
		FilesAdded:   scan.FilesAdded,
		FilesUpdated: scan.FilesUpdated,
		FilesDeleted: scan.FilesDeleted,
		ErrorMessage: scan.ErrorMessage,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create scan history: %w", err)
	}

	scan.ID = model.ID
	return nil
}

// UpdateScanHistory updates a scan history record
func (r *GormRepository) UpdateScanHistory(ctx context.Context, scan *domain.ScanResult) error {
	updates := map[string]interface{}{
		"completed_at":  scan.CompletedAt,
		"files_scanned": scan.FilesScanned,
		"files_added":   scan.FilesAdded,
		"files_updated": scan.FilesUpdated,
		"files_deleted": scan.FilesDeleted,
		"error_message": scan.ErrorMessage,
	}

	result := r.db.WithContext(ctx).Model(&ScanHistory{}).Where("id = ?", scan.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update scan history: %w", result.Error)
	}

	return nil
}

// GetLatestScan gets the latest scan for a library
func (r *GormRepository) GetLatestScan(ctx context.Context, libraryID uuid.UUID) (*domain.ScanResult, error) {
	var model ScanHistory
	if err := r.db.WithContext(ctx).Where("library_id = ?", libraryID).Order("started_at DESC").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No scan history is not an error
		}
		return nil, fmt.Errorf("failed to get latest scan: %w", err)
	}

	return r.toDomainScanResult(&model), nil
}

// CreateEpisode creates a new episode
func (r *GormRepository) CreateEpisode(ctx context.Context, episode *models.Episode) error {
	model := &Episode{
		MediaID:       episode.MediaID,
		SeasonNumber:  episode.SeasonNumber,
		EpisodeNumber: episode.EpisodeNumber,
		Title:         episode.Title,
		AirDate:       &episode.AirDate,
		Runtime:       episode.Duration / 60, // Convert seconds to minutes
		FilePath:      episode.Path,
		Status:        "available", // Default status
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create episode: %w", err)
	}

	episode.ID = model.ID
	episode.Added = model.CreatedAt
	return nil
}

// GetEpisode retrieves an episode by ID
func (r *GormRepository) GetEpisode(ctx context.Context, id uuid.UUID) (*models.Episode, error) {
	var model Episode
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("episode not found")
		}
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	return r.toDomainEpisode(&model), nil
}

// GetEpisodeByNumber retrieves an episode by media ID, season and episode number
func (r *GormRepository) GetEpisodeByNumber(ctx context.Context, mediaID uuid.UUID, season, episode int) (*models.Episode, error) {
	var model Episode
	if err := r.db.WithContext(ctx).Where("media_id = ? AND season_number = ? AND episode_number = ?", mediaID, season, episode).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("episode not found")
		}
		return nil, fmt.Errorf("failed to get episode by number: %w", err)
	}

	return r.toDomainEpisode(&model), nil
}

// ListEpisodesByMedia lists all episodes for a media item
func (r *GormRepository) ListEpisodesByMedia(ctx context.Context, mediaID uuid.UUID) ([]*models.Episode, error) {
	var items []Episode
	if err := r.db.WithContext(ctx).Where("media_id = ?", mediaID).Order("season_number, episode_number").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list episodes by media: %w", err)
	}

	episodes := make([]*models.Episode, len(items))
	for i := range items {
		episodes[i] = r.toDomainEpisode(&items[i])
	}

	return episodes, nil
}

// ListEpisodesBySeason lists all episodes for a specific season
func (r *GormRepository) ListEpisodesBySeason(ctx context.Context, mediaID uuid.UUID, season int) ([]*models.Episode, error) {
	var items []Episode
	if err := r.db.WithContext(ctx).Where("media_id = ? AND season_number = ?", mediaID, season).Order("episode_number").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list episodes by season: %w", err)
	}

	episodes := make([]*models.Episode, len(items))
	for i := range items {
		episodes[i] = r.toDomainEpisode(&items[i])
	}

	return episodes, nil
}

// UpdateEpisode updates an episode
func (r *GormRepository) UpdateEpisode(ctx context.Context, episode *models.Episode) error {
	updates := map[string]interface{}{
		"title":     episode.Title,
		"air_date":  episode.AirDate,
		"runtime":   episode.Duration / 60, // Convert seconds to minutes
		"file_path": episode.Path,
		"status":    "available", // Default status
	}

	result := r.db.WithContext(ctx).Model(&Episode{}).Where("id = ?", episode.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update episode: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("episode not found")
	}

	return nil
}

// DeleteEpisode deletes an episode
func (r *GormRepository) DeleteEpisode(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&Episode{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete episode: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("episode not found")
	}

	return nil
}

// CreateProvider creates a new metadata provider
func (r *GormRepository) CreateProvider(ctx context.Context, provider *domain.MetadataProviderConfig) error {
	// Encrypt the API key before storing
	encryptedKey, err := r.encryptor.Encrypt(provider.APIKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	model := &MetadataProvider{
		Name:         provider.Name,
		ProviderType: provider.ProviderType,
		APIKey:       encryptedKey,
		Enabled:      provider.Enabled,
		Priority:     provider.Priority,
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	provider.ID = model.ID
	provider.CreatedAt = model.CreatedAt
	provider.UpdatedAt = model.UpdatedAt
	return nil
}

// GetProvider retrieves a metadata provider by ID
func (r *GormRepository) GetProvider(ctx context.Context, id uuid.UUID) (*domain.MetadataProviderConfig, error) {
	var model MetadataProvider
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("provider not found")
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return r.toDomainProvider(&model), nil
}

// GetProviderByName retrieves a metadata provider by name
func (r *GormRepository) GetProviderByName(ctx context.Context, name string) (*domain.MetadataProviderConfig, error) {
	var model MetadataProvider
	if err := r.db.WithContext(ctx).First(&model, "name = ?", name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound("provider not found")
		}
		return nil, fmt.Errorf("failed to get provider by name: %w", err)
	}

	return r.toDomainProvider(&model), nil
}

// ListProviders lists metadata providers
func (r *GormRepository) ListProviders(ctx context.Context, enabled *bool, providerType *string) ([]*domain.MetadataProviderConfig, error) {
	query := r.db.WithContext(ctx)

	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	if providerType != nil && *providerType != "" {
		query = query.Where("provider_type = ?", *providerType)
	}

	var items []MetadataProvider
	if err := query.Order("priority DESC, name").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	providers := make([]*domain.MetadataProviderConfig, len(items))
	for i, model := range items {
		providers[i] = r.toDomainProvider(&model)
	}

	return providers, nil
}

// UpdateProvider updates a metadata provider
func (r *GormRepository) UpdateProvider(ctx context.Context, provider *domain.MetadataProviderConfig) error {
	// Encrypt the API key before storing
	encryptedKey, err := r.encryptor.Encrypt(provider.APIKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	updates := map[string]interface{}{
		"name":          provider.Name,
		"provider_type": provider.ProviderType,
		"api_key":       encryptedKey,
		"enabled":       provider.Enabled,
		"priority":      provider.Priority,
	}

	result := r.db.WithContext(ctx).Model(&MetadataProvider{}).Where("id = ?", provider.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update provider: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("provider not found")
	}

	return nil
}

// DeleteProvider deletes a metadata provider
func (r *GormRepository) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&MetadataProvider{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete provider: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return pkgerrors.NotFound("provider not found")
	}

	return nil
}

// Transaction support
func (r *GormRepository) BeginTx(ctx context.Context) (Repository, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	return &GormRepository{db: tx}, nil
}

func (r *GormRepository) Commit() error {
	return r.db.Commit().Error
}

func (r *GormRepository) Rollback() error {
	return r.db.Rollback().Error
}

// Helper methods to convert between database and domain models
func (r *GormRepository) toDomainLibrary(model *Library) *domain.Library {
	lib := &domain.Library{
		ID:           model.ID,
		Name:         model.Name,
		Path:         model.Path,
		Type:         model.MediaType,
		Enabled:      model.Enabled,
		ScanInterval: model.ScanInterval,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}

	if model.LastScanAt != nil {
		lib.LastScanAt = model.LastScanAt
	}

	return lib
}

func (r *GormRepository) toDomainMedia(model *MediaItem) *models.Media {
	media := &models.Media{
		ID:             model.ID,
		LibraryID:      model.LibraryID,
		Title:          model.Title,
		Type:           models.MediaType(model.MediaType),
		Path:           model.FilePath,
		Size:           model.FileSize,
		Duration:       model.Runtime * 60, // Convert minutes to seconds
		Resolution:     model.Resolution,
		Codec:          model.VideoCodec,
		Bitrate:        model.Bitrate,
		Added:          model.CreatedAt,
		Modified:       model.UpdatedAt,
		LastScanned:    model.UpdatedAt,
		Status:         model.Status,
		FilePath:       model.FilePath,
		FileSize:       model.FileSize,
		FileModifiedAt: model.FileModifiedAt,
		Description:    model.Description,
		Genres:         model.Genres,
		Tags:           model.Tags,
		TMDBID:         model.TMDBID,
		IMDBID:         model.IMDBID,
		TVDBID:         model.TVDBID,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}

	if model.ReleaseDate != nil {
		media.ReleaseDate = *model.ReleaseDate
	}

	return media
}

func (r *GormRepository) toDomainScanResult(model *ScanHistory) *domain.ScanResult {
	return &domain.ScanResult{
		ID:           model.ID,
		LibraryID:    model.LibraryID,
		StartedAt:    model.StartedAt,
		CompletedAt:  model.CompletedAt,
		FilesScanned: model.FilesScanned,
		FilesAdded:   model.FilesAdded,
		FilesUpdated: model.FilesUpdated,
		FilesDeleted: model.FilesDeleted,
		ErrorMessage: model.ErrorMessage,
	}
}

func (r *GormRepository) toDomainEpisode(model *Episode) *models.Episode {
	ep := &models.Episode{
		ID:            model.ID,
		MediaID:       model.MediaID,
		SeasonNumber:  model.SeasonNumber,
		EpisodeNumber: model.EpisodeNumber,
		Title:         model.Title,
		Path:          model.FilePath,
		Duration:      model.Runtime * 60, // Convert minutes to seconds
		Added:         model.CreatedAt,
	}

	if model.AirDate != nil {
		ep.AirDate = *model.AirDate
	}

	return ep
}

func (r *GormRepository) toDomainProvider(model *MetadataProvider) *domain.MetadataProviderConfig {
	// Decrypt the API key before returning
	decryptedKey, err := r.encryptor.Decrypt(model.APIKey)
	if err != nil {
		// Log error but don't fail - return empty key
		decryptedKey = ""
	}

	return &domain.MetadataProviderConfig{
		ID:           model.ID,
		Name:         model.Name,
		ProviderType: model.ProviderType,
		APIKey:       decryptedKey,
		Enabled:      model.Enabled,
		Priority:     model.Priority,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
