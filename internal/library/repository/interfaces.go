package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// LibraryRepository defines the interface for library data access.
type LibraryRepository interface {
	CreateLibrary(ctx context.Context, library *domain.Library) error
	GetLibrary(ctx context.Context, id uuid.UUID) (*domain.Library, error)
	GetLibraryByPath(ctx context.Context, path string) (*domain.Library, error)
	UpdateLibrary(ctx context.Context, library *domain.Library) error
	DeleteLibrary(ctx context.Context, id uuid.UUID) error
	ListLibraries(ctx context.Context, enabled *bool) ([]*domain.Library, error)
}

// MediaRepository defines the interface for media data access.
type MediaRepository interface {
	CreateMedia(ctx context.Context, media *models.Media) error
	GetMedia(ctx context.Context, id uuid.UUID) (*models.Media, error)
	GetMediaByPath(ctx context.Context, path string) (*models.Media, error)
	SearchMedia(
		ctx context.Context,
		query string,
		mediaType *string,
		status *string,
		libraryID *uuid.UUID,
		limit, offset int,
	) ([]*models.Media, error)
	UpdateMedia(ctx context.Context, media *models.Media) error
	DeleteMedia(ctx context.Context, id uuid.UUID) error
	ListMediaByLibrary(
		ctx context.Context,
		libraryID uuid.UUID,
		status *string,
		limit, offset int,
	) ([]*models.Media, error)
}

// EpisodeRepository defines the interface for episode data access.
type EpisodeRepository interface {
	CreateEpisode(ctx context.Context, episode *models.Episode) error
	GetEpisode(ctx context.Context, id uuid.UUID) (*models.Episode, error)
	GetEpisodeByNumber(ctx context.Context, mediaID uuid.UUID, season, episode int) (*models.Episode, error)
	ListEpisodesByMedia(ctx context.Context, mediaID uuid.UUID) ([]*models.Episode, error)
	ListEpisodesBySeason(ctx context.Context, mediaID uuid.UUID, season int) ([]*models.Episode, error)
	UpdateEpisode(ctx context.Context, episode *models.Episode) error
	DeleteEpisode(ctx context.Context, id uuid.UUID) error
}

// ScanRepository defines the interface for scan history data access.
type ScanRepository interface {
	CreateScanHistory(ctx context.Context, scan *domain.ScanResult) error
	UpdateScanHistory(ctx context.Context, scan *domain.ScanResult) error
	GetLatestScan(ctx context.Context, libraryID uuid.UUID) (*domain.ScanResult, error)
}

// MetadataProviderRepository defines the interface for metadata provider data access.
type MetadataProviderRepository interface {
	CreateProvider(ctx context.Context, provider *domain.MetadataProviderConfig) error
	GetProvider(ctx context.Context, id uuid.UUID) (*domain.MetadataProviderConfig, error)
	GetProviderByName(ctx context.Context, name string) (*domain.MetadataProviderConfig, error)
	ListProviders(ctx context.Context, enabled *bool, providerType *string) ([]*domain.MetadataProviderConfig, error)
	UpdateProvider(ctx context.Context, provider *domain.MetadataProviderConfig) error
	DeleteProvider(ctx context.Context, id uuid.UUID) error
}

// Repository aggregates all repository interfaces.
type Repository interface {
	LibraryRepository
	MediaRepository
	EpisodeRepository
	ScanRepository
	MetadataProviderRepository

	// Transaction support
	BeginTx(ctx context.Context) (Repository, error)
	Commit() error
	Rollback() error
}
