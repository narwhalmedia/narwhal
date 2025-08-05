package interfaces

import (
	"context"
	
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// Repository defines a generic repository interface
type Repository[T any] interface {
	// Get retrieves an entity by ID
	Get(ctx context.Context, id string) (*T, error)
	
	// List retrieves entities based on filter criteria
	List(ctx context.Context, filter Filter) ([]*T, error)
	
	// Create creates a new entity
	Create(ctx context.Context, entity *T) error
	
	// Update updates an existing entity
	Update(ctx context.Context, entity *T) error
	
	// Delete deletes an entity by ID
	Delete(ctx context.Context, id string) error
	
	// Count returns the number of entities matching the filter
	Count(ctx context.Context, filter Filter) (int64, error)
}

// Filter represents query filters
type Filter struct {
	Limit      int
	Offset     int
	OrderBy    string
	OrderDesc  bool
	Conditions map[string]interface{}
}

// UnitOfWork represents a transactional unit of work
type UnitOfWork interface {
	// Begin starts a new transaction
	Begin(ctx context.Context) error
	
	// Commit commits the transaction
	Commit() error
	
	// Rollback rolls back the transaction
	Rollback() error
	
	// MediaRepo returns the media repository within this transaction
	MediaRepo() MediaRepository
	
	// MetadataRepo returns the metadata repository within this transaction
	MetadataRepo() MetadataRepository
	
	// LibraryRepo returns the library repository within this transaction
	LibraryRepo() LibraryRepository
}

// MediaRepository defines media-specific repository operations
type MediaRepository interface {
	Repository[models.Media]
	
	// FindByPath finds media by file path
	FindByPath(ctx context.Context, path string) (*models.Media, error)
	
	// FindByLibrary finds all media in a library
	FindByLibrary(ctx context.Context, libraryID string) ([]*models.Media, error)
	
	// Search searches media by title or metadata
	Search(ctx context.Context, query string, filter Filter) ([]*models.Media, error)
}

// MetadataRepository defines metadata-specific repository operations
type MetadataRepository interface {
	Repository[models.Metadata]
	
	// FindByMediaID finds metadata by media ID
	FindByMediaID(ctx context.Context, mediaID string) (*models.Metadata, error)
	
	// FindByExternalID finds metadata by external ID (IMDB, TMDB, etc.)
	FindByExternalID(ctx context.Context, source, externalID string) (*models.Metadata, error)
}

// LibraryRepository defines library-specific repository operations
type LibraryRepository interface {
	Repository[models.Library]
	
	// FindByPath finds a library by path
	FindByPath(ctx context.Context, path string) (*models.Library, error)
	
	// FindAutoScanEnabled finds all libraries with auto-scan enabled
	FindAutoScanEnabled(ctx context.Context) ([]*models.Library, error)
}