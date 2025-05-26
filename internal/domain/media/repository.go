package media

import (
	"context"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/domain/specification"
)

// Repository defines the interface for media repositories
type Repository interface {
	// Save saves a media aggregate
	Save(ctx context.Context, aggregate Aggregate) error
	// FindByID finds a media aggregate by ID
	FindByID(ctx context.Context, id uuid.UUID) (Aggregate, error)
	// Delete deletes a media aggregate
	Delete(ctx context.Context, id uuid.UUID) error
}

// SeriesRepository defines the interface for series repositories
type SeriesRepository interface {
	// Save saves a series
	Save(ctx context.Context, series *Series) error
	// FindByID finds a series by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Series, error)
	// Delete deletes a series
	Delete(ctx context.Context, id uuid.UUID) error
	// FindByTitle finds a series by title
	FindByTitle(ctx context.Context, title string) (*Series, error)
	// FindAll returns all series
	FindAll(ctx context.Context) ([]*Series, error)
	// FindByStatus finds series by status
	FindByStatus(ctx context.Context, status Status) ([]*Series, error)
	// FindBySpecification finds series matching the specification
	FindBySpecification(ctx context.Context, spec specification.Specification) ([]*Series, error)
	// CountBySpecification counts series matching the specification
	CountBySpecification(ctx context.Context, spec specification.Specification) (int64, error)
}

// MovieRepository defines the interface for movie repositories
type MovieRepository interface {
	// Save saves a movie
	Save(ctx context.Context, movie *Movie) error
	// FindByID finds a movie by ID
	FindByID(ctx context.Context, id uuid.UUID) (*Movie, error)
	// Delete deletes a movie
	Delete(ctx context.Context, id uuid.UUID) error
	// FindByTitle finds a movie by title
	FindByTitle(ctx context.Context, title string) (*Movie, error)
	// FindAll returns all movies
	FindAll(ctx context.Context) ([]*Movie, error)
	// FindByStatus finds movies by status
	FindByStatus(ctx context.Context, status Status) ([]*Movie, error)
	// FindByGenre finds movies by genre
	FindByGenre(ctx context.Context, genre string) ([]*Movie, error)
	// FindByDirector finds movies by director
	FindByDirector(ctx context.Context, director string) ([]*Movie, error)
	// FindBySpecification finds movies matching the specification
	FindBySpecification(ctx context.Context, spec specification.Specification) ([]*Movie, error)
	// CountBySpecification counts movies matching the specification
	CountBySpecification(ctx context.Context, spec specification.Specification) (int64, error)
}

// EpisodeRepository defines the interface for episode repositories
type EpisodeRepository interface {
	Repository
	// FindBySeriesID finds episodes by series ID
	FindBySeriesID(ctx context.Context, seriesID uuid.UUID) ([]*Episode, error)
	// FindBySeason finds episodes by series ID and season number
	FindBySeason(ctx context.Context, seriesID uuid.UUID, seasonNumber int) ([]*Episode, error)
	// FindByStatus finds episodes by status
	FindByStatus(ctx context.Context, status Status) ([]*Episode, error)
	// FindBySpecification finds episodes matching the specification
	FindBySpecification(ctx context.Context, spec specification.Specification) ([]*Episode, error)
	// CountBySpecification counts episodes matching the specification
	CountBySpecification(ctx context.Context, spec specification.Specification) (int64, error)
}