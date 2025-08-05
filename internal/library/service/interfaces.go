package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/library/domain"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// LibraryServiceInterface defines the interface for library service operations
type LibraryServiceInterface interface {
	// Library operations
	CreateLibrary(ctx context.Context, library *domain.Library) error
	GetLibrary(ctx context.Context, id uuid.UUID) (*domain.Library, error)
	ListLibraries(ctx context.Context, enabled *bool) ([]*domain.Library, error)
	UpdateLibrary(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*domain.Library, error)
	DeleteLibrary(ctx context.Context, id uuid.UUID) error
	ScanLibrary(ctx context.Context, id uuid.UUID) error

	// Media operations
	GetMedia(ctx context.Context, id uuid.UUID) (*models.Media, error)
	SearchMedia(ctx context.Context, query string, mediaType *string, status *string, libraryID *uuid.UUID, limit, offset int) ([]*models.Media, error)
	UpdateMedia(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*models.Media, error)
	DeleteMedia(ctx context.Context, id uuid.UUID) error
	ListMediaByLibrary(ctx context.Context, libraryID uuid.UUID, status *string, limit, offset int) ([]*models.Media, error)

	// Scan operations
	GetLatestScan(ctx context.Context, libraryID uuid.UUID) (*domain.ScanResult, error)
}

// Ensure LibraryService implements the interface
var _ LibraryServiceInterface = (*LibraryService)(nil)
