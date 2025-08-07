package repository

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/narwhalmedia/narwhal/pkg/encryption"
	"github.com/narwhalmedia/narwhal/pkg/models"
	"github.com/narwhalmedia/narwhal/pkg/repository"
)

// GormRepository implements the repository interfaces using GORM.
type GormRepository struct {
	db        *gorm.DB
	encryptor *encryption.Encryptor
}

// NewGormRepository creates a new GORM repository.
func NewGormRepository(db *gorm.DB) (*GormRepository, error) {
	encryptionKey := os.Getenv("NARWHAL_ENCRYPTION_KEY")
	if encryptionKey == "" {
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

// CreateLibrary creates a new library.
func (r *GormRepository) CreateLibrary(ctx context.Context, library *models.Library) error {
	return repository.Create(ctx, r.db, library)
}

// GetLibrary retrieves a library by ID.
func (r *GormRepository) GetLibrary(ctx context.Context, id uuid.UUID) (*models.Library, error) {
	return repository.FindByID[models.Library](ctx, r.db, id)
}

// GetLibraryByPath retrieves a library by path.
func (r *GormRepository) GetLibraryByPath(ctx context.Context, path string) (*models.Library, error) {
	return repository.FindOneBy[models.Library](ctx, r.db, "path = ?", path)
}

// UpdateLibrary updates a library.
func (r *GormRepository) UpdateLibrary(ctx context.Context, library *models.Library) error {
	return repository.Update(ctx, r.db, library)
}

// DeleteLibrary deletes a library.
func (r *GormRepository) DeleteLibrary(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.Library](ctx, r.db, id)
}

// ListLibraries lists all libraries.
func (r *GormRepository) ListLibraries(ctx context.Context, enabled *bool) ([]*models.Library, error) {
	query := r.db.WithContext(ctx)
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	var items []*models.Library
	if err := query.Order("name").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list libraries: %w", err)
	}
	return items, nil
}

// CreateMedia creates a new media item.
func (r *GormRepository) CreateMedia(ctx context.Context, media *models.Media) error {
	return repository.Create(ctx, r.db, media)
}

// GetMedia retrieves a media item by ID.
func (r *GormRepository) GetMedia(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	return repository.FindByID[models.Media](ctx, r.db, id)
}

// GetMediaByPath retrieves a media item by file path.
func (r *GormRepository) GetMediaByPath(ctx context.Context, path string) (*models.Media, error) {
	return repository.FindOneBy[models.Media](ctx, r.db, "file_path = ?", path)
}

// SearchMedia searches for media items.
func (r *GormRepository) SearchMedia(
	ctx context.Context,
	query string,
	mediaType *string,
	status *string,
	libraryID *uuid.UUID,
	limit, offset int,
) ([]*models.Media, error) {
	q := r.db.WithContext(ctx).Model(&models.Media{})

	if query != "" {
		q = q.Where("title ILIKE ? OR original_title ILIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if mediaType != nil && *mediaType != "" {
		q = q.Where("media_type = ?", *mediaType)
	}
	if status != nil && *status != "" {
		q = q.Where("status = ?", *status)
	}
	if libraryID != nil {
		q = q.Where("library_id = ?", *libraryID)
	}

	var items []*models.Media
	if err := q.Order("title").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to search media: %w", err)
	}
	return items, nil
}

// UpdateMedia updates a media item.
func (r *GormRepository) UpdateMedia(ctx context.Context, media *models.Media) error {
	return repository.Update(ctx, r.db, media)
}

// DeleteMedia deletes a media item.
func (r *GormRepository) DeleteMedia(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.Media](ctx, r.db, id)
}

// ListMediaByLibrary lists media items by library.
func (r *GormRepository) ListMediaByLibrary(
	ctx context.Context,
	libraryID uuid.UUID,
	status *string,
	limit, offset int,
) ([]*models.Media, error) {
	q := r.db.WithContext(ctx).Model(&models.Media{}).Where("library_id = ?", libraryID)

	if status != nil && *status != "" {
		q = q.Where("status = ?", *status)
	}

	var items []*models.Media
	if err := q.Order("title").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list media by library: %w", err)
	}
	return items, nil
}

// CreateScanHistory creates a new scan history record.
func (r *GormRepository) CreateScanHistory(ctx context.Context, scan *models.ScanHistory) error {
	return repository.Create(ctx, r.db, scan)
}

// UpdateScanHistory updates a scan history record.
func (r *GormRepository) UpdateScanHistory(ctx context.Context, scan *models.ScanHistory) error {
	return repository.Update(ctx, r.db, scan)
}

// GetLatestScan gets the latest scan for a library.
func (r *GormRepository) GetLatestScan(ctx context.Context, libraryID uuid.UUID) (*models.ScanHistory, error) {
	var model models.ScanHistory
	if err := r.db.WithContext(ctx).Where("library_id = ?", libraryID).Order("started_at DESC").First(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

// CreateEpisode creates a new episode.
func (r *GormRepository) CreateEpisode(ctx context.Context, episode *models.Episode) error {
	return repository.Create(ctx, r.db, episode)
}

// GetEpisode retrieves an episode by ID.
func (r *GormRepository) GetEpisode(ctx context.Context, id uuid.UUID) (*models.Episode, error) {
	return repository.FindByID[models.Episode](ctx, r.db, id)
}

// GetEpisodeByNumber retrieves an episode by media ID, season and episode number.
func (r *GormRepository) GetEpisodeByNumber(
	ctx context.Context,
	mediaID uuid.UUID,
	season, episode int,
) (*models.Episode, error) {
	return repository.FindOneBy[models.Episode](ctx, r.db, "media_id = ? AND season_number = ? AND episode_number = ?", mediaID, season, episode)
}

// ListEpisodesByMedia lists all episodes for a media item.
func (r *GormRepository) ListEpisodesByMedia(ctx context.Context, mediaID uuid.UUID) ([]*models.Episode, error) {
	var items []*models.Episode
	if err := r.db.WithContext(ctx).Where("media_id = ?", mediaID).Order("season_number, episode_number").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list episodes by media: %w", err)
	}
	return items, nil
}

// ListEpisodesBySeason lists all episodes for a specific season.
func (r *GormRepository) ListEpisodesBySeason(
	ctx context.Context,
	mediaID uuid.UUID,
	season int,
) ([]*models.Episode, error) {
	var items []*models.Episode
	if err := r.db.WithContext(ctx).Where("media_id = ? AND season_number = ?", mediaID, season).Order("episode_number").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list episodes by season: %w", err)
	}
	return items, nil
}

// UpdateEpisode updates an episode.
func (r *GormRepository) UpdateEpisode(ctx context.Context, episode *models.Episode) error {
	return repository.Update(ctx, r.db, episode)
}

// DeleteEpisode deletes an episode.
func (r *GormRepository) DeleteEpisode(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.Episode](ctx, r.db, id)
}

// CreateProvider creates a new metadata provider.
func (r *GormRepository) CreateProvider(ctx context.Context, provider *models.MetadataProvider) error {
	encryptedKey, err := r.encryptor.Encrypt(provider.APIKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}
	provider.APIKey = encryptedKey
	return repository.Create(ctx, r.db, provider)
}

// GetProvider retrieves a metadata provider by ID.
func (r *GormRepository) GetProvider(ctx context.Context, id uuid.UUID) (*models.MetadataProvider, error) {
	provider, err := repository.FindByID[models.MetadataProvider](ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	decryptedKey, err := r.encryptor.Decrypt(provider.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}
	provider.APIKey = decryptedKey
	return provider, nil
}

// GetProviderByName retrieves a metadata provider by name.
func (r *GormRepository) GetProviderByName(ctx context.Context, name string) (*models.MetadataProvider, error) {
	provider, err := repository.FindOneBy[models.MetadataProvider](ctx, r.db, "name = ?", name)
	if err != nil {
		return nil, err
	}
	decryptedKey, err := r.encryptor.Decrypt(provider.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}
	provider.APIKey = decryptedKey
	return provider, nil
}

// ListProviders lists metadata providers.
func (r *GormRepository) ListProviders(
	ctx context.Context,
	enabled *bool,
	providerType *string,
) ([]*models.MetadataProvider, error) {
	query := r.db.WithContext(ctx)

	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	if providerType != nil && *providerType != "" {
		query = query.Where("provider_type = ?", *providerType)
	}

	var items []*models.MetadataProvider
	if err := query.Order("priority DESC, name").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	for _, p := range items {
		decryptedKey, err := r.encryptor.Decrypt(p.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key for provider %s: %w", p.Name, err)
		}
		p.APIKey = decryptedKey
	}
	return items, nil
}

// UpdateProvider updates a metadata provider.
func (r *GormRepository) UpdateProvider(ctx context.Context, provider *models.MetadataProvider) error {
	encryptedKey, err := r.encryptor.Encrypt(provider.APIKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}
	provider.APIKey = encryptedKey
	return repository.Update(ctx, r.db, provider)
}

// DeleteProvider deletes a metadata provider.
func (r *GormRepository) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.MetadataProvider](ctx, r.db, id)
}

// Transaction support.
func (r *GormRepository) BeginTx(ctx context.Context) (Repository, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	encryptor, err := encryption.NewEncryptor(os.Getenv("NARWHAL_ENCRYPTION_KEY"))
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor for transaction: %w", err)
	}
	return &GormRepository{db: tx, encryptor: encryptor}, nil
}

func (r *GormRepository) Commit() error {
	return r.db.Commit().Error
}

func (r *GormRepository) Rollback() error {
	return r.db.Rollback().Error
}
